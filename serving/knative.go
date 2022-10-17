package serving

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"

	uuid "github.com/google/uuid"
	"github.com/ndau/dao-voting-setup/dal"
	"github.com/ndau/dao-voting-setup/models"
	logger "github.com/ndau/go-logger"
	"github.com/ndau/go-ndau"
)

const (
	maxKafkaHops = "222"
)

// KnClient -
type KnClient struct {
	// Optional: logging
	Log logger.Logger
}

// NewKnClient -
func NewKnClient(cfg *models.Config, loggers ...logger.Logger) (knc *KnClient, err error) {
	// Attach an optional logger
	var log logger.Logger
	if len(loggers) > 0 {
		log = loggers[0]
	} else {
		log = &logger.NoopLogger{}
	}

	return &KnClient{
		// Optional: logging
		Log: log,
	}, nil
}

//Run knative function
func (k *KnClient) Run(ctx context.Context, repo dal.Repo, cfg *models.Config) error {
	k.Log.Info("Starting knative run...")
	k.Listen(ctx, repo, cfg)
	return nil
}

// Listen ...
func (k *KnClient) Listen(ctx context.Context, repo dal.Repo, cfg *models.Config) {
	trackingNumber := uuid.New().String()

	handler := func(w http.ResponseWriter, r *http.Request) {
		thisContext := context.WithValue(ctx, "tracking_number", trackingNumber)

		k.Log.Infof("%s | Start processing knative request", trackingNumber)
		fmt.Printf("%+v\n", r)
		switch r.Method {
		case "POST":
			if body, err := ioutil.ReadAll(r.Body); err != nil {
				k.Log.Errorf("%s | Failed to read request body", trackingNumber, err)
			} else {
				var data models.Data
				if err := json.Unmarshal(body, &data); err != nil {
					k.Log.Errorf("%s | Failed to unmarshal request data", trackingNumber, err)
				} else {
					err := k.ProcessEvent(thisContext, &data, repo, cfg)
					if err != nil {
						k.Log.Errorf("%s | Failed to process the request: %v", trackingNumber, err)
					} else {
						k.Log.Infof("%s | Finish")
					}
				}
			}
		default:
			k.Log.Errorf("%s | Sorry, only POST method are supported", trackingNumber)
		}
	}

	port := "8080"
	http.HandleFunc("/", handler)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		k.Log.Errorf("%s | Failed to listening on the port %d: %v", trackingNumber, port, err)
	}

	k.Log.Infof("knative is listening on port %d", port)
}

// ProcessEvent ...
func (k *KnClient) ProcessEvent(ctx context.Context, data *models.Data, repo dal.Repo, cfg *models.Config) error {
	trackingNumber, _ := ctx.Value("tracking_number").(string)

	k.Log.Infof("%s | Start processing event...", trackingNumber)
	// The “currency seat date” is the date at which the account’s balance first reached 1,000 ndau
	// after the most recent time it was below 1,000. If an account reached 1,000 ndau on 1/1/21 and
	// hasn’t gone below 1,000 since, then that’s its currency seat date. If that same account’s balance
	// dropped to 999 ndau yesterday but came back above 1,000 today, its currency seat date is today.
	// Accounts with fewer than 1,000 ndau in them have no currency seat date.

	network := data.Network
	baseURL := data.NodeAPI

	k.Log.Infof("%s | Network/NodeAPI: %s/%s", trackingNumber, network, baseURL)

	// Create the NdauAPI client
	defaultClient := http.DefaultClient
	conn, err := ndau.New(defaultClient, &ndau.NdauConfig{
		Network: network,
		NodeAPI: baseURL,
	}, k.Log)
	if err != nil {
		k.Log.Errorf("%s | Failed to instantiate ndau client to the network %s. Error = %s", trackingNumber, network, err.Error())
		return err
	}

	// Get non-duplicated account list
	cache, err := k.cacheBuilder(ctx, data, repo, conn)
	if err != nil {
		k.Log.Errorf("%s | Failed to build existing accounts cache", trackingNumber)
		return err
	}

	// Get account balances and currency seat dates
	accountList, total, err := k.watcher(ctx, data, cfg, cache, repo, conn)
	if err != nil {
		k.Log.Errorf("%s | Failed to run diff with the account cache", trackingNumber)
		return err
	}

	// Order by a currency seat date: the oldest first
	sort.Slice(accountList, func(i, j int) bool {
		return accountList[i].CurrencySeatDate.Before(accountList[j].CurrencySeatDate)
	})

	// Enforce maximum 3000 seats
	// accountList = accountList[:3000]

	k.Log.Infof("%s | Got %d acounts with currency seat date", trackingNumber, len(accountList))

	for idx, d := range accountList {
		if d.Balance >= 1000 {
			fmt.Println(idx, d.CurrencySeatDate, d.Balance)
		}
	}

	// Compute voting power for each seated account
	if err = k.updateVote(ctx, accountList, total, repo, conn); err != nil {
		k.Log.Errorf("%s | Failed to update account votings", trackingNumber)
	}

	k.Log.Infof("%s | Done", trackingNumber)

	return nil
}

func (k *KnClient) cacheBuilder(ctx context.Context, data *models.Data, repo dal.Repo, conn *ndau.Ndau) (models.Cached, error) {
	trackingNumber, _ := ctx.Value("tracking_number").(string)

	var params interface{}

	// Create a set of distinguished accounts
	var void struct{}
	cache := models.Cached{}

	// limit is the number of accounts in a single query -- this is limited by the
	// blockchain API and so we have to do a set of requests to get all the data
	limit := data.Limit
	after := data.StartAfterKey

	for {
		if after == "" {
			break
		} else {
			k.Log.Infof("%s | Get all accounts after '%v'", trackingNumber, after)
			input, _ := json.Marshal(ndau.AccountListReq{
				Limit: limit,
				After: after,
			})

			json.Unmarshal(input, &params)
		}
		api := "/account/list"
		res, err := conn.GetDataWithContext(ctx, api, params)
		if err != nil {
			k.Log.Errorf("%s | Failed to get accounts: %s", trackingNumber, err.Error())
			return nil, err
		}

		var r ndau.AccountListResp
		if err = json.Unmarshal(res, &r); err != nil {
			k.Log.Errorf("%s | Failed to unmarshall the response: %s", trackingNumber, err.Error())
			break
		}
		// debug
		// k.Log.Infof("%s | Got %d acounts. The next one would be after %s", trackingNumber, len(r.Accounts), r.NextAfter)
		if len(r.Accounts) > 0 {
			for _, address := range r.Accounts {
				cache[address] = void
			}
		}

		after = r.NextAfter
	}

	k.Log.Infof("%s | Cached successfully %d accounts", trackingNumber, len(cache))

	return cache, nil
}

func (k *KnClient) watcher(ctx context.Context, data *models.Data, cfg *models.Config, cache models.Cached, repo dal.Repo, conn *ndau.Ndau) (votingList []ndau.Account, totalNdau int, err error) {
	trackingNumber, _ := ctx.Value("tracking_number").(string)

	count := 0
	numberOfAccounts := len(cache)
	addresses := []string{}

	for address, _ := range cache {
		addresses = append(addresses, address)
		count++
		numberOfAccounts--

		if count == 100 || numberOfAccounts == 0 {
			// Now sort the slice
			sort.Strings(addresses)

			// Update account balance
			if accounts, total_balance, err := k.updateBalance(ctx, addresses, conn); err == nil {
				totalNdau = totalNdau + total_balance
				votingList = append(votingList, accounts...)
			} else {
				k.Log.Errorf("%s | Failed to update account balances from address: %s. Error = %s", trackingNumber, addresses[0], err.Error())
				return nil, 0, err
			}

			count = 0
			addresses = []string{}
		}
	}

	return votingList, totalNdau, nil
}

func (k *KnClient) updateBalance(ctx context.Context, addresses []string, conn *ndau.Ndau) (accounts []ndau.Account, total_balance int, err error) {
	trackingNumber, _ := ctx.Value("tracking_number").(string)

	total_balance = 0

	// Read accounts details
	var params interface{}

	input, _ := json.Marshal(addresses)
	json.Unmarshal(input, &params)
	api := "/account/accounts"
	res, err := conn.PostDataWithContext(ctx, api, params)
	if err != nil {
		k.Log.Errorf("%s | Failed to get account voting list: %s", trackingNumber, err.Error())
		return nil, total_balance, err
	}

	var r ndau.AccountResp
	if err = json.Unmarshal(res, &r); err != nil {
		k.Log.Errorf("%s | Failed to unmarshall account detail response: %s", trackingNumber, err.Error())
		return nil, total_balance, err
	}

	for address, val := range r {
		if val.CurrencySeatDate.Year() >= 2016 {
			account := ndau.Account{
				Id:               address,
				CurrencySeatDate: val.CurrencySeatDate,
				Balance:          val.Balance,
			}
			accounts = append(accounts, account)
			total_balance = total_balance + val.Balance
		} else {
			total_balance = total_balance + val.Balance
		}
	}

	k.Log.Infof("%s | Updated balances for %d accounts", trackingNumber, len(accounts))

	return accounts, total_balance, nil
}

func (k *KnClient) updateVote(ctx context.Context, votingList []ndau.Account, total_balance int, repo dal.Repo, conn *ndau.Ndau) error {
	trackingNumber, _ := ctx.Value("tracking_number").(string)

	k.Log.Infof("%s | Get current price and total Ndau...", trackingNumber)

	api := "/price/current"
	res, err := conn.GetDataWithContext(ctx, api, nil)
	if err != nil {
		k.Log.Errorf("%s | Failed to get total Ndau: %s", trackingNumber, err.Error())
		return err
	}

	r := ndau.CurrentPriceResp{}
	if err = json.Unmarshal(res, &r); err != nil {
		k.Log.Errorf("%s | Failed to unmarshall current price response: %s", trackingNumber, err.Error())
		return err
	}

	k.Log.Infof("%s | Total Ndau = %d", trackingNumber, r.TotalNdau)
	if total_balance != r.TotalNdau {
		k.Log.Warnf("%s | Unmatched total Ndau: %d", trackingNumber, total_balance)
	}

	// Now let's compute the voting power for each seated account

	votes := []models.VotingSetup{}

	// There are 9,000,000 votes in total
	//   - One third of the votes are assigned equally to each currency seat.
	//   - One third of the votesare assigned proportionally to each address based on its share of all ndau in circulation
	//   - And the final third are assigned equally to each of the three oldest currency seat addresses
	noOfCurrencySeat := len(votingList)
	for i := 0; i < noOfCurrencySeat; i++ {
		vote := models.VotingSetup{
			Address:          votingList[i].Id,
			CurrencySeatDate: votingList[i].CurrencySeatDate,
			Votes:            3000000 * (1.0/float64(noOfCurrencySeat) + float64(votingList[i].Balance)) / float64(r.TotalNdau),
		}
		votes = append(votes, vote)
	}

	votes[0].Votes = votes[0].Votes + 1000000
	votes[1].Votes = votes[1].Votes + 1000000
	votes[2].Votes = votes[2].Votes + 1000000

	k.Log.Infof("%s | Start updating %d account votings...", trackingNumber, len(votingList))

	if err := repo.UpsertVotingList(ctx, votes); err != nil {
		k.Log.Errorf("%s | Failed to insert to send_file_log table. Error: %v", trackingNumber, err)
	}

	return nil
}
