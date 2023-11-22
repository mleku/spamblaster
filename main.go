package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/mleku/spamblaster/pkg/creator"
	"github.com/mleku/spamblaster/pkg/logger"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/mleku/spamblaster/pkg/strfry"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/spf13/viper"
)

func decodePub(pubKey string) string {
	if strings.Contains(pubKey, "npub") {
		if _, v, err := nip19.Decode(pubKey); err == nil {
			pubKey = v.(string)
		}
	}
	return pubKey
}

func queryRelay(oldRelay *creator.Relay) (*creator.Relay, error) {

	var relay creator.Relay

	// example spamblaster config
	url := "http://127.0.0.1:3000/api/sconfig/relays/clkklcjon000wgh31mcgbut40"

	body, err := os.ReadFile("./spamblaster.cfg")
	if err != nil {
		log.Err(fmt.Sprintf("unable to read config file: %v", err))
	} else {
		url = strings.TrimSuffix(string(body), "\n")
	}

	rClient := http.Client{
		Timeout: time.Second * 10,
	}

	var req *http.Request
	req, err = http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Err(err.Error())
		return oldRelay, err
	}

	res, getErr := rClient.Do(req)
	if getErr != nil {
		log.Err(getErr.Error())
		return oldRelay, err
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

	jsonErr := json.Unmarshal(body, &relay)
	if jsonErr != nil {
		log.Err("json not unmarshaled")
		return oldRelay, jsonErr
	}

	return &relay, nil
}

// initialise logger
var log = logger.NewLogger()

type influxdbConfig struct {
	Url         string `mapstructure:"INFLUXDB_URL"`
	Token       string `mapstructure:"INFLUXDB_TOKEN"`
	Org         string `mapstructure:"INFLUXDB_ORG"`
	Bucket      string `mapstructure:"INFLUXDB_BUCKET"`
	Measurement string `mapstructure:"INFLUXDB_MEASUREMENT"`
}

func main() {
	var err error
	var reader = bufio.NewReader(os.Stdin)
	var output = bufio.NewWriter(os.Stdout)
	// close logger properly
	defer func() {
		err = log.Close()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr,
				"error closing logger: %s\n", err)
		}
	}()
	// close output writer properly
	defer func() {
		err = output.Flush()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr,
				"error flushing output writer: %s\n", err)
		}
	}()

	var relay *creator.Relay
	relay, err = queryRelay(relay)
	if err != nil {
		log.Err("there was an error fetching relay, using cache or nil")
	}

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for {
			<-ticker.C
			relay, err = queryRelay(relay)
			if err != nil {
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

		var e strfry.Event
		if err := json.Unmarshal([]byte(input), &e); err != nil {
			panic(err)
		}

		result := strfry.Result{
			ID:     e.Event.ID,
			Action: "accept",
		}

		allowMessage := false
		if relay.DefaultMessagePolicy {
			allowMessage = true
		}
		badResp := ""

		// moderation retroactive delete
		if e.Event.Kind == strfry.MemoryHole {
			isModAction := false
			for _, m := range relay.Moderators {
				usePub := decodePub(m.User.Pubkey)
				if usePub == e.Event.PubKey {
					isModAction = true
				}
			}
			if relay.Owner.Pubkey == e.Event.PubKey {
				isModAction = true
			}
			if isModAction {
				log.Err(fmt.Sprintf("1984 (memory hole) request from %s>",
					e.Event.PubKey))
				// perform deletion of a single event
				// grab the event id
				thisReason := ""
				thisEvent := ""
				for _, x := range e.Event.Tags {
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
						e.Event.PubKey, thisEvent, thisReason,
					))
					// shell out
					filter := fmt.Sprintf("{\"ids\": [\"%s\"]}", thisEvent)
					cmd := exec.Command("/app/strfry", "delete", "--filter",
						filter)
					out, err := cmd.Output()
					if err != nil {
						log.Err(fmt.Sprintln("could not run command: ", err))
					}
					log.Err(fmt.Sprintln("strfry command output: ",
						string(out)))

					// cmd.Run()
				}

				// if modaction is for a pubkey, post back to the API for
				// block_list pubkey also delete the events related to this
				// pubkey
				thisPubkey := ""
				for _, x := range e.Event.Tags {
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
					log.Info(fmt.Sprintf("received 1984 (memory hole) from mod: %s, block and delete public key <%s>, reason: %s",
						e.Event.PubKey, thisPubkey, thisReason))
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

		// pubkeys logic: false is deny, true is allow
		if !relay.DefaultMessagePolicy {
			// relay is in whitelist pubkey mode, only allow these pubkeys to post
			for _, k := range relay.AllowList.ListPubkeys {
				if strings.Contains(k.Pubkey, "npub") {
					if _, v, err := nip19.Decode(k.Pubkey); err == nil {
						pub := v.(string)
						if strings.Contains(e.Event.PubKey, pub) {
							log.Err("allowing whitelist for public key: " + k.Pubkey)
							allowMessage = true
						}
					} else {
						log.Err("error decoding public key: " + k.Pubkey + " " + err.Error())
					}
				}

				if strings.Contains(e.Event.PubKey, k.Pubkey) {
					log.Info("allowing whitelist for public key: " + k.Pubkey)
					allowMessage = true
				}
			}
		}

		// keywords logic
		if relay.AllowList.ListKeywords != nil && len(relay.AllowList.ListKeywords) >= 1 && !relay.DefaultMessagePolicy {
			// relay has whitelist keywords, allow messages matching any of
			// these keywords to post, deny messages that don't.
			//
			// todo: what about if they're allow_listed pubkey? (currently this
			//  would allow either)
			for _, k := range relay.AllowList.ListKeywords {
				dEvent := strings.ToLower(e.Event.Content)
				dKeyword := strings.ToLower(k.Keyword)
				if strings.Contains(dEvent, dKeyword) {
					log.Info("allowing for keyword: " + k.Keyword)
					allowMessage = true
				}
			}
		}

		if relay.BlockList.ListPubkeys != nil &&
			len(relay.BlockList.ListPubkeys) >= 1 {

			// relay is in blacklist pubKey mode, mark bad
			for _, k := range relay.BlockList.ListPubkeys {
				if strings.Contains(k.Pubkey, "npub") {
					if _, v, err := nip19.Decode(k.Pubkey); err == nil {
						pub := v.(string)
						if strings.Contains(e.Event.PubKey, pub) {
							log.Info("rejecting for public key: " + k.Pubkey)
							badResp = "blocked public key " + k.Pubkey + " reason: " + k.Reason
							allowMessage = false
						}
					} else {
						log.Err("error decoding public key: " + k.Pubkey + " " + err.Error())
					}
				}
				if strings.Contains(e.Event.PubKey, k.Pubkey) {
					log.Info("rejecting for public key: " + k.Pubkey)
					badResp = "blocked public key " + k.Pubkey + " reason: " + k.Reason
					allowMessage = false
				}
			}
		}

		if relay.BlockList.ListKeywords != nil && len(relay.BlockList.ListKeywords) >= 1 {
			// relay has blacklist keywords, deny messages matching any of these keywords to post
			for _, k := range relay.BlockList.ListKeywords {
				dEvent := strings.ToLower(e.Event.Content)
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
		_, err = output.WriteString(fmt.Sprintf("%s\n", r))
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr,
				"error writing output: %s\n", err)
		}
		err = output.Flush()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr,
				"error flushing Writer: %s\n", err)
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
					"kind":  fmt.Sprintf("%d", e.Event.Kind),
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
