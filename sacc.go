package main

import (
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	DSA "ipfs"
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
}

type ContractSignature struct {
	Signature map[string]string
}

type SignatureContract struct {
	Contract          Contract
	ContractSignature ContractSignature
}

type Log struct {
	Address   string
	TimeStamp int64
    AntiCheatResultAddress []string
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

	if fn == "submit" {
		err = submit(stub, args)
	}

	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success([]byte(result))
}

// args[0]:id
// args[1]:file location
// args[2]:private key
func submit(stub shim.ChaincodeStubInterface, args []string) error {
	if len(args) != 3 {
		return "", fmt.Errorf("Incorrect arguments. Expecting 3 value")
	}
	sc, e1 := stub.GetState(args[0] + "_contract")
	if e1 != nil {
		return e1
	}
	if sc == "" {
		return nil
	}
	var signatureContract SignatureContract
	e2 := json.Unmarshal(sc, &signatureContract)
	if e2 != nil {
		return e2
	}
    //if all people have signed contract
	antiCheatIds := sc.Contract.AntiCheatIds
    if len(antiCheatIds)+2 != len(sc.ContractSignature) {
        return nil
    }
	//#######
	log := Log{Address: args[1]}
	signature, e3 := DSA.Sign(log, args[2])
	if e3 != nil {
		return e3
	}
	contractSignature := map[string]string{args[0]: signature}
	mediaLogSubmit := MediaLogSubmit{Log: log, ContractSignature: contractSignature}
	mls, _ := json.Marshal(mediaLogSubmit)
	//######
	for _, id := range antiCheatIds {
		stub.PutState(id+"_log", mls)
	}
	stub.PutState(args[0]+"_contract", "")
	return nil
}

func confirm(stub shim.ChaincodeStubInterface, args []string) error {
    stub.GetState
	return nil
}

// main function starts up the chaincode in the container during instantiate
func main() {
	if err := shim.Start(new(SimpleAsset)); err != nil {
		fmt.Printf("Error starting SimpleAsset chaincode: %s", err)
	}
}