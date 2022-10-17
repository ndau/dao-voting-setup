# Ndau DAO voting setup

## Run
1. Update the `config.yaml` file under the folder `config` to the values of your environment
1. Run:    
```sh
  NDAU_CONFIG_NAME=config NDAU_CONFIG_PATH=./config go run main.go
```
## Test on mainnet
```sh
curl -v "http://localhost:8080" \
-H "Content-Type: application/json" \
-d '{"Network":"mainnet","NodeAPI":"<your-node-api:3030>","Limit":100,"StartAfterKey": "-"}'

```