package main

import (
	"fmt"
	"time"
	"github.com/hyperledger/fabric/core/chaincode/lib/cid"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"utils/DSA"
)

// SimpleAsset implements a simple chaincode to manage an asset
type SimpleAsset struct {
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
	AntiCheatNum           int64
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

	if fn == "mediaSubmit" {
		err = submit(stub, args)
	} else if fn == "contractList" {
		result, err = contractList(stub, args)
	}

	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success([]byte(result))
}

func ContractInit(args []string, timeStamp int64) Contract {
	var contract Contract

	contract.AdvertiserId = args[0]
	contract.MediaId = args[1]
	contract.AntiCheatIds = strings.Split(args[2], ",")
	contract.PaymentThreshold = args[3]
	contract.PaymentAmountMedia = args[4]
	contract.PaymentAmountAntiCheat = args[5]
	contract.AntiCheatShareType = args[6]
	contract.AntiCheatPriority = strings.Split(args[7], ",")
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
* 6: AntiCheat_Share_Type
* 7: AntiCheat_Priority
* 8: PrivateKey
*/
func ContractGenerator(stub shim.ChaincodeStubInterface, args []string) error {
	if len(args) != 9 {
		return "", fmt.Errorf("Incorrect arguments. Expecting 3 value")
	}

	id, err := cid.GetID(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("Could not Get ID, err %s", err))
	}
	time_stamp := time.Now().Unix()
	key = fmt.Sprintf("%s_%s_%s_%d", id, args[0], args[1], timeStamp)

	contract := ContractInit(args[:7], timeStamp)

	var signatureContract SignatureContract
	signatureContract.Contract = contract

	contractJson, _ := json.Marshal(contract)
	signature, err := DSA.Sign(string(contractJson), args[8])
	if err != nil {
		return err
	}
	signatureContract.ContractSignature[id] = signature

	signatureContractJson, _ := json.Marshal(signatureContract)
	stub.PutState(key, string(signatureContractJson))

	stub.PutState(args[0] + "_contract", key)
	antiCheatIds :=  strings.Split(args[1], ",")
	for index, value := range antiCheatIds {
		stub.PutState(value + "_contract", key)
	}
}

// get contract msg according to contract id
func getContract(stub shim.ChaincodeStubInterface, args []string) (string, error) {
	sc, err := stub.GetState(contractId)
	if err != nil {
		return "", err
	}
	var signatureContract SignatureContract
	err = json.Unmarshal(sc, &signatureContract)
	if err != nil {
		return "", err
	}
}

// contractList get history contracts of media or anticheat
func contractList(stub shim.ChaincodeStubInterface, args []string) (string, error) {
	id, err := cid.GetID(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("Could not Get ID, err %s", err))
	}
	it, err := stub.GetHistoryForKey(id + "_contract")
	if err != nil {
		return "", err
	}

	result, err := getHistoryListResult(it)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func getHistoryListResult(resultsIterator shim.HistoryQueryIteratorInterface) ([]byte, error) {

	defer resultsIterator.Close()
	// buffer is a JSON array containing QueryRecords
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		item, _ := json.Marshal(queryResponse)
		buffer.Write(item)
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")
	fmt.Printf("queryResult:\n%s\n", buffer.String())
	return buffer.Bytes(), nil
}

// args[0]:contract id
// args[1]:file location
// args[2]:private key
func mediaSubmit(stub shim.ChaincodeStubInterface, args []string) error {
	if len(args) != 3 {
		return "", fmt.Errorf("Incorrect arguments. Expecting 3 value")
	}
	contractId := args[0]
	fileLocation := args[1]
	privateKey := args[2]
	id, err := cid.GetID(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("Could not Get ID, err %s", err))
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
	antiCheatIds := sc.Contract.AntiCheatIds
	if len(antiCheatIds)+2 != len(sc.ContractSignature) {
		return nil
	}
	//#######
	log := Log{Address: fileLocation, AntiCheatNum: len(antiCheatIds)}
	logJson, _ := json.Marshal(log)
	signature, err := DSA.Sign(string(logJson), privateKey)
	if err != nil {
		return err
	}
	contractSignature := map[string][]byte{id: signature}
	mediaLogSubmit := MediaLogSubmit{Log: log, ContractSignature: contractSignature}
	mls, _ := json.Marshal(mediaLogSubmit)
	stub.PutState(contractId+"_log", string(mls))
	//######
	for _, id := range antiCheatIds {
		stub.PutState(id+"_log", contractId+"_log")
	}
	return nil
}

func logList(stub shim.ChaincodeStubInterface, args []string) (string, error) {
	id, err := cid.GetID(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("Could not Get ID, err %s", err))
	}
	it, err := stub.GetHistoryForKey(id + "_log")
	if err != nil {
		return "", err
	}

	result, err := getHistoryListResult(it)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// args[0]:log id
// args[1]:filepath
// args[2]:private key
func anticheatConfirm(stub shim.ChaincodeStubInterface, args []string) error {
	if len(args) != 3 {
		return "", fmt.Errorf("Incorrect arguments. Expecting 3 value")
	}
	logId := args[0]
	fileLocation := args[1]
	privateKey := args[2]
	id, err := cid.GetID(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("Could not Get ID, err %s", err))
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
	sig, err := DSA.Sign(logJson, privateKey)
	if err != nil {
		return err
	}
	mediaLogSubmit.ContractSignature[id] = sig

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
