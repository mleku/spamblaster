package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/relaytools/spamblaster/pkg/creator"
	"github.com/relaytools/spamblaster/pkg/logger"
	"github.com/relaytools/spamblaster/pkg/strfry"
	"github.com/relaytools/spamblaster/pkg/util"
	"github.com/spf13/viper"
	"mleku.online/git/replicatr/pkg/nostr/kind"
)

func queryRelay(oldRelay *creator.Relay) (r *creator.Relay, e error) {

	r = &creator.Relay{}

	// example spamblaster config
	url := "http://127.0.0.1:3000/api/sconfig/relays/clkklcjon000wgh31mcgbut40"

	var body []byte
	body, e = os.ReadFile("./spamblaster.cfg")
	if e != nil {
		log.Err(fmt.Sprintf("unable to read config file: %v", e))
	} else {
		url = strings.TrimSuffix(string(body), "\n")
	}

	rClient := http.Client{
		Timeout: time.Second * 10,
	}

	var req *http.Request
	req, e = http.NewRequest(http.MethodGet, url, nil)
	if e != nil {
		log.Err(e.Error())
		return oldRelay, e
	}

	res, getErr := rClient.Do(req)
	if getErr != nil {
		log.Err(getErr.Error())
		return oldRelay, e
	}

	if res.Body != nil {
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr,
					"error closing response Body: %s\n", err)
			}
		}(res.Body)
	}

	body, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		log.Err(readErr.Error())
		return oldRelay, readErr
	}

	jsonErr := json.Unmarshal(body, r)
	if jsonErr != nil {
		log.Err("json not unmarshaled")
		return oldRelay, jsonErr
	}

	return r, e
}

// initialise logger
var log = logger.NewLogger("spamblaster")

type influxdbConfig struct {
	Url         string `mapstructure:"INFLUXDB_URL"`
	Token       string `mapstructure:"INFLUXDB_TOKEN"`
	Org         string `mapstructure:"INFLUXDB_ORG"`
	Bucket      string `mapstructure:"INFLUXDB_BUCKET"`
	Measurement string `mapstructure:"INFLUXDB_MEASUREMENT"`
}

func main() {
	var e error
	reader := bufio.NewReader(os.Stdin)
	output := bufio.NewWriter(os.Stdout)

	log.Info("starting up spamblaster")

	// close output writer properly
	defer func() {
		e = output.Flush()
		if e != nil {
			_, _ = fmt.Fprintf(os.Stderr,
				"error flushing output writer: %s\n", e)
		}
	}()

	var relay *creator.Relay
	relay, e = queryRelay(relay)
	if e != nil {
		log.Err("there was an error fetching relay, using cache or nil")
	}

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for {
			<-ticker.C
			relay, e = queryRelay(relay)
			if e != nil {
				log.Err("there was an error fetching relay, using cache or nil")
			}
		}
	}()

	// InfluxDB optional config loading
	viper.AddConfigPath("/usr/local/etc")
	viper.SetConfigName(".spamblaster.env")
	viper.SetConfigType("env")
	influxEnabled := true
	var iConfig *influxdbConfig
	if err := viper.ReadInConfig(); err != nil {
		log.Info(fmt.Sprint("Warn: error reading influxdb config file /usr/local/etc/.spamblaster.env\n",
			err))
		influxEnabled = false
	}
	// Viper unmarshals the loaded env variables into the struct
	if err := viper.Unmarshal(&iConfig); err != nil {
		log.Err(fmt.Sprint("Warn: unable to decode influxdb config into struct\n",
			err))
		influxEnabled = false
	}

	log.Err(fmt.Sprintf("Info: influxdb: %t\n", influxEnabled))

	var client influxdb2.Client
	var writeAPI api.WriteAPI

	if influxEnabled {
		// INFLUX INIT
		client = influxdb2.NewClientWithOptions(iConfig.Url, iConfig.Token,
			influxdb2.DefaultOptions().SetBatchSize(20))
		// Get non-blocking write client
		writeAPI = client.WriteAPI(iConfig.Org, iConfig.Bucket)
	}

	for {
		var input, _ = reader.ReadString('\n')

		var ev strfry.Event
		if e = json.Unmarshal([]byte(input), &e); e != nil {
			// this should not happen, just skip to the next
			continue
		}

		result := strfry.Result{
			ID:     ev.ID,
			Action: "accept",
		}

		allowMessage := false
		if relay.DefaultMessagePolicy {
			allowMessage = true
		}
		badResp := ""

		// moderation retroactive delete
		if ev.Kind == kind.MemoryHole {
			isModAction := false
			for _, m := range relay.Moderators {
				var usePub string
				usePub, e = util.DecodePub(m.User.Pubkey)
				if usePub == ev.PubKey {
					isModAction = true
				}
			}
			if relay.Owner.Pubkey == ev.PubKey {
				isModAction = true
			}
			if isModAction {
				log.Err(fmt.Sprintf("1984 (memory hole) request from %s>",
					ev.PubKey))
				// perform deletion of a single event
				// grab the event id
				thisReason := ""
				thisEvent := ""
				for _, x := range ev.Tags {
					if x[0] == "e" {
						thisEvent = x[1]
						if len(x) == 3 {
							thisReason = x[2]
						}
					}
				}

				if thisEvent != "" {
					log.Info(fmt.Sprintf(
						"received 1984 (memory hole) from mod: %s, delete event <%s>, reason: %s",
						ev.PubKey, thisEvent, thisReason,
					))
					// shell out
					filter := fmt.Sprintf("{\"ids\": [\"%s\"]}", thisEvent)
					cmd := exec.Command("/app/strfry", "delete", "--filter",
						filter)
					var out []byte
					out, e = cmd.Output()
					if e != nil {
						log.Err(fmt.Sprintln("could not run command: ", e))
					}
					log.Err(fmt.Sprintln("strfry command output: ",
						string(out)))

					// cmd.Run()
				}

				// if modaction is for a pubkey, post back to the API for
				// block_list pubkey also delete the events related to this
				// pubkey
				thisPubkey := ""
				for _, x := range ev.Tags {
					if x[0] == "p" {
						thisPubkey = x[1]
						if len(x) == 3 {
							thisReason = x[2]
						}
					}
				}

				// event should be blank if we're getting a report about just a
				// pubkey
				if thisPubkey != "" && thisEvent == "" {
					log.Info(fmt.Sprintf("received 1984 (memory hole) from mod: %s,"+
						" block and delete public key <%s>, reason: %s",
						ev.PubKey, thisPubkey, thisReason))
					// shell out
					filter := fmt.Sprintf("{\"authors\": [\"%s\"]}", thisPubkey)
					cmd := exec.Command("/app/strfry", "delete", "--filter",
						filter)
					out, err := cmd.Output()
					if err != nil {
						log.Err(fmt.Sprintln("could not run command: ", err))
					}
					log.Info(fmt.Sprintln("strfry command output: ",
						string(out)))
					// cmd.Run()
					// TODO: call to api
				}

			}
		}

		var pub string

		// pubkeys logic: false is deny, true is allow
		if !relay.DefaultMessagePolicy {
			// relay is in whitelist pubkey mode, only allow these pubkeys to post
			for _, k := range relay.AllowList.ListPubkeys {
				if strings.HasPrefix(k.Pubkey, "npub") {
					if pub, e = util.NpubToHex(k.Pubkey); e == nil {
						if strings.Contains(ev.PubKey, pub) {
							log.Err("allowing whitelist for public key: " + k.Pubkey)
							allowMessage = true
						}
					} else {
						log.Err("error decoding public key: " + k.Pubkey + " " + e.Error())
					}
				}

				if strings.Contains(ev.PubKey, k.Pubkey) {
					log.Info("allowing whitelist for public key: " + k.Pubkey)
					allowMessage = true
				}
			}
		}

		// keywords logic
		if len(relay.AllowList.ListKeywords) > 0 && !relay.DefaultMessagePolicy {
			// relay has whitelist keywords, allow messages matching any of
			// these keywords to post, deny messages that don't.
			//
			// todo: what about if they're allow_listed pubkey? (currently this
			//  would allow either)
			for _, k := range relay.AllowList.ListKeywords {
				dEvent := strings.ToLower(ev.Content)
				dKeyword := strings.ToLower(k.Keyword)
				if strings.Contains(dEvent, dKeyword) {
					log.Info("allowing for keyword: " + k.Keyword)
					allowMessage = true
				}
			}
		}

		if len(relay.BlockList.ListPubkeys) > 0 {

			// relay is in blacklist pubKey mode, mark bad
			for _, k := range relay.BlockList.ListPubkeys {
				if strings.Contains(k.Pubkey, "npub") {
					if pub, e = util.NpubToHex(k.Pubkey); e == nil {
						if strings.Contains(ev.PubKey, pub) {
							log.Info("rejecting for public key: " + k.Pubkey)
							badResp = "blocked public key " + k.Pubkey + " reason: " + k.Reason
							allowMessage = false
						}
					} else {
						log.Err("error decoding public key: " + k.Pubkey + " " + e.Error())
					}
				}
				if strings.Contains(ev.PubKey, k.Pubkey) {
					log.Info("rejecting for public key: " + k.Pubkey)
					badResp = "blocked public key " + k.Pubkey + " reason: " + k.Reason
					allowMessage = false
				}
			}
		}

		if len(relay.BlockList.ListKeywords) > 0 {
			// relay has blacklist keywords, deny messages matching any of these keywords to post
			for _, k := range relay.BlockList.ListKeywords {
				dEvent := strings.ToLower(ev.Content)
				dKeyword := strings.ToLower(k.Keyword)
				if strings.Contains(dEvent, dKeyword) {
					log.Info("rejecting for keyword: " + k.Keyword)
					badResp = "blocked. " + k.Keyword + " reason: " + k.Reason
					allowMessage = false
				}
			}
		}

		if !allowMessage {
			result.Action = "reject"
			result.Msg = badResp
		}

		r, _ := json.Marshal(result)
		_, e = output.WriteString(fmt.Sprintf("%s\n", r))
		if e != nil {
			_, _ = fmt.Fprintf(os.Stderr,
				"error writing output: %s\n", e)
		}
		e = output.Flush()
		if e != nil {
			_, _ = fmt.Fprintf(os.Stderr,
				"error flushing Writer: %s\n", e)
		}

		if influxEnabled {
			blocked, allowed := 0, 1
			// if filter mode is deny
			if !allowMessage {
				blocked, allowed = 1, 0
			}

			p := influxdb2.NewPoint(
				iConfig.Measurement,
				map[string]string{
					"kind":  fmt.Sprintf("%d", ev.Kind),
					"relay": relay.ID,
				},
				map[string]interface{}{
					"event":   1,
					"blocked": blocked,
					"allowed": allowed,
				},
				time.Now())
			// write asynchronously
			writeAPI.WritePoint(p)
		}
	}
}
