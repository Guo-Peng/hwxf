package main

import (
	"fmt"
	"time"
	"strings"
	"encoding/json"
	"github.com/hyperledger/fabric/core/chaincode/lib/cid"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"utils/DSA"
    "io/ioutil"
)

// SimpleAsset implements a simple chaincode to manage an asset
type SimpleAsset struct {
}

type Account struct {
	Type string `json:"type"`
	Credit string `json:"credit"`
	Assets string `json:"assets"`
	PublicKey string `json:"public_key"`
}

type Contract struct {
	AdvertiserId           string
	MediaId                string
	AntiCheatIds           []string
	PaymentThreshold       string
	PaymentAmountMedia     string
	PaymentAmountAntiCheat string
	AntiCheatShareType     string
	AntiCheatPriority      []string
	TimeStamp			   int64
}

type ContractSignature struct {
	Signature map[string][]byte
}

type SignatureContract struct {
	Contract          Contract
	ContractSignature ContractSignature
}

type Log struct {
	Address                string
	TimeStamp              int64
	AntiCheatResultAddress []string
	AntiCheatNum           int
}

type MediaLogSubmit struct {
	Log               Log
	ContractSignature ContractSignature
}

func (t *SimpleAsset) Init(stub shim.ChaincodeStubInterface) peer.Response {
	return shim.Success(nil)
}

func (t *SimpleAsset) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	fn, args := stub.GetFunctionAndParameters()
	var result string
	var err error

	if fn == "setAccount" {
		result, err = setAccount(stub, args)
	} else if fn == "generatorContract" {
		err = generatorContract(stub, args)
	} else if fn == "mediaSubmit" {
		err = mediaSubmit(stub, args)
	} else if fn == "getContract" {
		result, err = getContractList(stub, args)
	} else if fn == "getContractList" {
		result, err = getContractList(stub, args)
	} else if fn == "getLogList" {
		result, err = getLogList(stub, args)
	}
	
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success([]byte(result))
}
/* ---------------------链码区域---------------------------*/
/*
* 0: Credit
* 1: Assets
* 2: PublicKey
*/
func setAccount(stub shim.ChaincodeStubInterface, args []string) (string, error) {

	if len(args) != 3 {
		return "", fmt.Errorf("Incorrect number of arguments. Expecting 4")
	}

	id, err := cid.GetID(stub)
	if err != nil {
		return "", fmt.Errorf(fmt.Sprintf("Could not Get ID, err %s", err))
	}
	mspid, err := cid.GetMSPID(stub)
	if err != nil {
		return "", fmt.Errorf(fmt.Sprintf("Could not Get MSP ID, err %s", err))
	}
    key,err := ioutil.ReadFile(args[2])
    if err != nil {
        return "",err
    }
	fmt.Printf("Id:\n%s\n", id)
	fmt.Printf("Type:\n%s\n", mspid)
	fmt.Printf("Credit:\n%s\n", args[0])
	fmt.Printf("Assets:\n%s\n", args[1])
	fmt.Printf("PublicKey:\n%s\n", string(key))

	var account = Account{Type: id, Credit: args[0], Assets: args[1], PublicKey: string(key)}

	accountAsBytes, _ := json.Marshal(account)
	stub.PutState(id, accountAsBytes)

	return id, nil
}

func getAccountPublicKey(stub shim.ChaincodeStubInterface, id string) (string, error) {
	accountAsBytes,err := stub.GetState(id)
	if err!=nil{
		return "", err
	}
	var account Account;
	err = json.Unmarshal(accountAsBytes,&account)
    if err!=nil{
        return "", err
    }
	return account.PublicKey, nil
}

func initContract(args []string, timeStamp int64, advertiserId string) Contract {
	var contract Contract

	contract.AdvertiserId = advertiserId
	contract.MediaId = args[0]
	contract.AntiCheatIds = strings.Split(args[1], ",")
	contract.PaymentThreshold = args[2]
	contract.PaymentAmountMedia = args[3]
	contract.PaymentAmountAntiCheat = args[4]
	contract.AntiCheatShareType = args[5]
	contract.AntiCheatPriority = strings.Split(args[6], ",")
	contract.TimeStamp = timeStamp
	return contract
}

/*
* 0: Media_Id
* 1: AntiCheat_Ids
* 2: Payment_Threshold
* 3: Payment_Amount_Media
* 4: Payment_Amount_AntiCheat
* 5: AntiCheat_Share_Type
* 6: AntiCheat_Priority
* 7: PrivateKey
*/
func generatorContract(stub shim.ChaincodeStubInterface, args []string) error {
	if len(args) != 9 {
		return fmt.Errorf("Incorrect arguments. Expecting 9 value")
	}

	id, err := cid.GetID(stub)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Could not Get ID, err %s", err))
	}
	timeStamp := time.Now().Unix()
	key := fmt.Sprintf("%s_%s_%s_%d", id, args[0], args[1], timeStamp)

	contract := initContract(args[:6], timeStamp, id)

	var signatureContract SignatureContract
	signatureContract.Contract = contract

	contractJson, _ := json.Marshal(contract)
	signature, err := DSA.Sign(string(contractJson), args[7])
	if err != nil {
		return err
	}
	signatureContract.ContractSignature.Signature[id] = signature

	signatureContractJson, _ := json.Marshal(signatureContract)
	stub.PutState(key, []byte(signatureContractJson))

	stub.PutState(args[0] + "_contract", []byte(key))
	antiCheatIds :=  strings.Split(args[1], ",")
	for _, value := range antiCheatIds {
		stub.PutState(value + "_contract", []byte(key))
	}
	return nil
}

// get contract msg according to contract id
func getContract(stub shim.ChaincodeStubInterface, contractId string) (string, error) {
	sc, err := stub.GetState(contractId)
	if err != nil {
		return "", err
	}
	return string(sc),nil
}

// getContractList get history contracts of media or anticheat
func getContractList(stub shim.ChaincodeStubInterface, args []string) (string, error) {
	id, err := cid.GetID(stub)
	if err != nil {
		return "", fmt.Errorf(fmt.Sprintf("Could not Get ID, err %s", err))
	}
	it, err := stub.GetHistoryForKey(id + "_contract")
	if err != nil {
		return "", err
	}

	resultList :=getHistoryListResult(it)
	return strings.Join(resultList, "\n"), nil
}

func getHistoryListResult(resultsIterator shim.HistoryQueryIteratorInterface) []string {

	defer resultsIterator.Close()

    s:= make([]string, 0, 10)
	// bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			continue
		}
		item, _ := json.Marshal(queryResponse)
        s=append(s,string(item))
	}
    return s
}

// args[0]:contract id
// args[1]:file location
// args[2]:private key
func mediaSubmit(stub shim.ChaincodeStubInterface, args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("Incorrect arguments. Expecting 3 value")
	}
	contractId := args[0]
	fileLocation := args[1]
	privateKey := args[2]
	id, err := cid.GetID(stub)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Could not Get ID, err %s", err))
	}
	sc, err := stub.GetState(contractId)
	if err != nil {
		return err
	}
	var signatureContract SignatureContract
	err = json.Unmarshal(sc, &signatureContract)
	if err != nil {
		return err
	}
	//if all people have signed contract
	antiCheatIds := signatureContract.Contract.AntiCheatIds
	if len(antiCheatIds)+2 != len(signatureContract.ContractSignature.Signature) {
		return fmt.Errorf("Could not submit, at Least one AntiCheatOrg not signed.")
	}
	//#######
	log := Log{Address: fileLocation, AntiCheatNum: len(antiCheatIds)}
	logJson, _ := json.Marshal(log)
	signature, err := DSA.Sign(string(logJson), privateKey)
	if err != nil {
		return err
	}
	contractSignature := ContractSignature{Signature: map[string][]byte{id: signature}}
	mediaLogSubmit := MediaLogSubmit{Log: log, ContractSignature: contractSignature}
	mls, _ := json.Marshal(mediaLogSubmit)
	stub.PutState(contractId+"_log", mls)
	//######
	for _, id := range antiCheatIds {
		stub.PutState(id+"_log", []byte(contractId+"_log"))
	}
	return nil
}

func getLogList(stub shim.ChaincodeStubInterface, args []string) (string, error) {
	id, err := cid.GetID(stub)
	if err != nil {
		return "", fmt.Errorf(fmt.Sprintf("Could not Get ID, err %s", err))
	}
	it, err := stub.GetHistoryForKey(id + "_log")
	if err != nil {
		return "", err
	}

	resultList := getHistoryListResult(it)
	return strings.Join(resultList, "\n"), nil
}

// args[0]:log id
// args[1]:filepath
// args[2]:private key
func anticheatConfirm(stub shim.ChaincodeStubInterface, args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("Incorrect arguments. Expecting 3 value")
	}
	logId := args[0]
	fileLocation := args[1]
	privateKey := args[2]
	id, err := cid.GetID(stub)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Could not Get ID, err %s", err))
	}
	msl, err := stub.GetState(logId)
	var mediaLogSubmit MediaLogSubmit
	err = json.Unmarshal(msl, &mediaLogSubmit)
	if err != nil {
		return err
	}
	//TODO DSA.Verify(privateKey)

	//anticheat Sign
	logJson, _ := json.Marshal(mediaLogSubmit.Log)
	sig, err := DSA.Sign(string(logJson), privateKey)
	if err != nil {
		return err
	}
	mediaLogSubmit.ContractSignature.Signature[id] = sig

	//put filelocation
	mediaLogSubmit.Log.AntiCheatResultAddress = append(mediaLogSubmit.Log.AntiCheatResultAddress, fileLocation)

	//if all have signed
	if mediaLogSubmit.Log.AntiCheatNum == len(mediaLogSubmit.Log.AntiCheatResultAddress) {
		fmt.Println("jiesuan")
        //TODO jiesuan
	}
	return nil
}

// main function starts up the chaincode in the container during instantiate
func main() {
	if err := shim.Start(new(SimpleAsset)); err != nil {
		fmt.Printf("Error starting SimpleAsset chaincode: %s", err)
	}
}
