package main

import (
    "encoding/json"
    "errors"
    "fmt"
    "github.com/hyperledger/fabric/core/chaincode/lib/cid"
    "github.com/hyperledger/fabric/core/chaincode/shim"
    "github.com/hyperledger/fabric/protos/peer"
    "os/exec"
    "strconv"
    "strings"
    "time"
    "utils/DSA"
)

const (
    RIGHT_CREDIT = 1 //each right judgement add 1 credit
    WRONG_CREDIT = 9 //each wrong judgement decrease 9 credit
)

// SimpleAsset implements a simple chaincode to manage an asset
type SimpleAsset struct {
}

type Account struct {
    Type      string `json:"type"`
    Credit    string `json:"credit"`
    Assets    string `json:"assets"`
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
    TimeStamp              int64
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
        result, err = getContract(stub, args[0])
    } else if fn == "getContractList" {
        result, err = getContractList(stub, args)
    } else if fn == "getLogList" {
        result, err = getLogList(stub, args)
    } else if fn == "advertiserMediaAntiConfirm" {
        err = advertiserMediaAntiConfirm(stub, args)
    } else if fn == "settleAccount" {
        err = settleAccount(stub, args)
    } else if fn == "getAllConfirmContractKey" {
        result, err = getAllConfirmContractKey(stub)
    } else if fn == "advertiserChargeGet" {
        err = advertiserChargeGet(stub, args)
    }

    if err != nil {
        return shim.Error(err.Error())
    }
    return shim.Success([]byte(result))
}

/* ---------------------链码区域---------------------------*/
/*
* 0: Type
* 1: Credit
* 2: Assets
* 3: PublicKey
 */
func setAccount(stub shim.ChaincodeStubInterface, args []string) (string, error) {

    if len(args) != 4 {
        return "", fmt.Errorf("Incorrect number of arguments. Expecting 4")
    }

    id, err := cid.GetID(stub)
    if err != nil {
        return "", fmt.Errorf(fmt.Sprintf("Could not Get ID, err %s", err))
    }
    fmt.Printf("Id:\n%s\n", id)
    fmt.Printf("Type:\n%s\n", args[0])
    fmt.Printf("Credit:\n%s\n", args[1])
    fmt.Printf("Assets:\n%s\n", args[2])
    fmt.Printf("PublicKey:\n%s\n", args[3])

    var account = Account{Type: args[0], Credit: args[1], Assets: args[2], PublicKey: args[3]}

    accountAsBytes, _ := json.Marshal(account)
    stub.PutState(id, accountAsBytes)

    return id, nil
}

func getAccountPublicKey(stub shim.ChaincodeStubInterface, id string) (string, error) {
    accountAsBytes, err := stub.GetState(id)
    if err != nil {
        return "", err
    }
    var account Account
    err = json.Unmarshal(accountAsBytes, &account)
    if err != nil {
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

func advertiserChargeGet(stub shim.ChaincodeStubInterface, args []string) error {
    if len(args) != 1 {
        return fmt.Errorf("Incorrect arguments. Expecting 1 value")
    }

    timePaymentByte, err := stub.GetState(args[0] + "_freeze")
    if err != nil {
        return err
    }

    timePayment := strings.Split(string(timePaymentByte), "_")
    if len(timePayment) != 2{
        return fmt.Errorf("timePayment format error: %s" , string(timePaymentByte))
    }

    timeStamp, err1 := strconv.Atoi(args[0])
    payment, err2 := strconv.Atoi(args[1])
    if err1 != nil || err2 != nil {
        return fmt.Errorf("timePayment format error: %s" , string(timePaymentByte))
    }

    if timeStamp < time.Now().Unix(){
        return fmt.Errorf("time is not up for your money: %d", timeStamp)
    }

    var account Account
    err = json.Unmarshal(ac, &account)
    if err != nil {
        return err
    }
    Account.Assets += payment

    accountAsBytes, _ := json.Marshal(account)
    stub.PutState(id, accountAsBytes)
    return nil
}

/*
* 0: advertiser id
* 1: payment
* 2: contractKey
*/
func advertiserCharge(stub shim.ChaincodeStubInterface, advertiserId string, paymentStr string, contractKey string) error {
    payment, err := strconv.Atoi(paymentStr)
    if err != nil {
        return err
    }

    ac, err := stub.GetState(advertiserId)
    if err != nil {
        return err
    }

    var account Account
    err = json.Unmarshal(ac, &account)
    if err != nil {
        return err
    }

    if account.Assets < payment {
        return fmt.Errorf("advertiser has not enough Assets")
    }
    Account.Assets -= payment

    accountAsBytes, _ := json.Marshal(account)
    stub.PutState(id, accountAsBytes)

    timeStamp := time.Now().Unix()
    stub.PutState(contractKey + "_freeze", []byte(fmt.Sprintf("%d_%d", timeStamp + 86400*7, payment)))
    return nil
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
    
    // 冻结合约金额
    err := advertiserCharge(stub, id, args[3], key)
    if err != nil {
        return err
    }

    signatureContract.ContractSignature.Signature[id] = signature
    signatureContractJson, _ := json.Marshal(signatureContract)
    stub.PutState(key, []byte(signatureContractJson))

    stub.PutState(id+"_confirm", []byte(key))
    stub.PutState(args[0]+"_confirm", []byte(key))
    antiCheatIds := strings.Split(args[1], ",")
    for _, value := range antiCheatIds {
        stub.PutState(value+"_confirm", []byte(key))
    }
    return nil
}

func getAllConfirmContractKey(stub shim.ChaincodeStubInterface) (string, error) {
    id, err := cid.GetID(stub)
    if err != nil {
        return "", fmt.Errorf(fmt.Sprintf("Could not Get ID, err %s", err))
    }

    it, err := stub.GetHistoryForKey(id + "_confirm")
    if err != nil {
        return "", err
    }

    resultList := getHistoryListResult(it)
    return strings.Join(resultList, "\n"), nil
}


/*
* 0: privateKey path
* 1: contractKey
*/
func advertiserMediaAntiConfirm(stub shim.ChaincodeStubInterface, args []string) error {
    if len(args) != 2 {
        return fmt.Errorf("Incorrect arguments. Expecting 2 value")
    }

    id, err := cid.GetID(stub)
    if err != nil {
        return fmt.Errorf(fmt.Sprintf("Could not Get ID, err %s", err))
    }

    ac, err := stub.GetState(id)
    if err != nil {
        return err
    }

    var account Account
    err = json.Unmarshal(ac, &account)
    if err != nil {
        return err
    }

    sc, err := stub.GetState(args[1])
    var signatureContract SignatureContract
    err = json.Unmarshal(sc, &signatureContract)
    if err != nil {
        return err
    }

    if account.Type == "Advertiser" {
        if len(signatureContract.ContractSignature.Signature) != len(signatureContract.Contract.AntiCheatIds)+2 {
            return nil
        }
    }

    for k, v := range signatureContract.ContractSignature.Signature {
        publicKey, err := getAccountPublicKey(stub, k)
        if err != nil {
            return err
        }

        contractJson, _ := json.Marshal(signatureContract.Contract)
        valid, err := DSA.Verify(string(contractJson), v, publicKey)
        if !valid {
            return fmt.Errorf(fmt.Sprintf("verify id %s failed", k))
        }
    }

    if account.Type == "Advertiser" {
        stub.PutState(signatureContract.Contract.MediaId+"_contract", contractKey)
        for _, value := range signatureContract.Contract.AntiCheatIds {
            stub.PutState(value+"_contract", contractKey)
        }
    } else {
        contractJson, _ := json.Marshal(signatureContract.Contract)
        signature, err := DSA.Sign(string(contractJson), args[0])
        if err != nil {
            return err
        }

        signatureContract.ContractSignature.Signature[id] = signature
        signatureContractJson, _ := json.Marshal(signatureContract)
        stub.PutState(string(contractKey), []byte(signatureContractJson))
    }

    return nil
}

// get contract msg according to contract id
func getContract(stub shim.ChaincodeStubInterface, contractId string) (string, error) {
    sc, err := stub.GetState(contractId)
    if err != nil {
        return "", err
    }
    return string(sc), nil
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

    resultList := getHistoryListResult(it)
    return strings.Join(resultList, "\n"), nil
}

func getHistoryListResult(resultsIterator shim.HistoryQueryIteratorInterface) []string {

    defer resultsIterator.Close()

    s := make([]string, 0, 10)
    // bArrayMemberAlreadyWritten := false
    for resultsIterator.HasNext() {
        queryResponse, err := resultsIterator.Next()
        if err != nil {
            continue
        }
        item, _ := json.Marshal(queryResponse)
        s = append(s, string(item))
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
    logJson, err := json.Marshal(mediaLogSubmit.Log)
    //TODO DSA.Verify(privateKey)
    for id, sig := range mediaLogSubmit.ContractSignature.Signature {
        acc, _ := stub.GetState(id)
        var account Account
        err = json.Unmarshal(acc, &account)
        if err != nil {
            return err
        }
        valid, err := DSA.Verify(string(logJson), sig, account.PublicKey)
        if err != nil {
            return err
        }
        if valid == false {
            return errors.New("not valid")
        }
    }
    //anticheat Sign
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

//args[0]: contractId
//args[1]: account-fileAddress map
func settleAccount(stub shim.ChaincodeStubInterface, args []string) error { //To Do: verify with public key
    scAsByte, err := stub.GetState(args[0])
    var sc Contract
    err = json.Unmarshal(scAsByte, &sc)
    if err != nil {
        return err
    }
    antiCheatIds := sc.AntiCheatIds
    antiCheatPriorityString := sc.AntiCheatPriority
    //transfer string into float64
    var antiCheatPriorityFloat = make([]float64, len(antiCheatPriorityString))
    for i := 0; i < len(antiCheatPriorityString); i++ {
        priority, err := strconv.ParseFloat(antiCheatPriorityString[i], 64)
        if err != nil {
            return err
        }
        antiCheatPriorityFloat[i] = priority
    }
    antiCheatAddressMap, err := getAddressMap(args[1]) //get address map from string
    if err != nil {
        return err
    }
    var antiCheatResults = make([][]float64, len(antiCheatIds))
    for i := 0; i < len(antiCheatIds); i++ {
        antiCheatResult, err := getAntiCheatResult(antiCheatAddressMap[antiCheatIds[i]])
        if err != nil {
            return err
        }
        antiCheatResults[i] = antiCheatResult
    }
    //count right and wrong judgement for each antiCheat
    var countArray = make([][2]int, len(antiCheatResults))
    for j := 0; j < len(antiCheatResults[0]); j++ {
        var sum float64
        for i := 0; i < len(antiCheatResults); i++ {
            sum += antiCheatResults[i][j] * antiCheatPriorityFloat[i]
        }
        for i := 0; i < len(antiCheatResults); i++ {
            if (sum >= 0 && antiCheatResults[i][j] == 1) || (sum < 0 && antiCheatResults[i][j] == -1) {
                countArray[i][0]++ //countArray[i][0] counts the right num
            } else {
                countArray[i][1]++ //countArray[i][1] counts the wrong num
            }
        }
    }
    return calculateMoneyAndCredit(stub, countArray, antiCheatIds)
}

func getAddressMap(addressStr string) (map[string]string, error) {
    strs := strings.Split(addressStr, ",")
    var addressMap = make(map[string]string, 0)
    for _, str := range strs {
        address := strings.Split(str, "\t")
        if len(address) < 2 {
            return nil, fmt.Errorf("address format error")
        }
        addressMap[address[0]] = address[1]
    }
    return addressMap, nil
}

//get file using ipfs
func getAntiCheatResult(address string) ([]float64, error) {
    if address == "" {
        return nil, fmt.Errorf("Incorrect arguments. Expecting Address as string")
    }
    cmd := "curl " + address
    output, err := exec.Command("sh", "-c", cmd).Output()
    if err != nil {
        return nil, err
    }
    strs := strings.Split(string(output), "\n")
    var result = make([]float64, len(strs))
    for i := 0; i < len(strs); i++ {
        num, err := strconv.ParseFloat(strings.Split(strs[i], "\t")[1], 64)
        if err != nil {
            return nil, err
        }
        result[i] = num
    }
    return result, nil
}

func calculateMoneyAndCredit(stub shim.ChaincodeStubInterface, countArray [][2]int, antiCheatIds []string) error {
    var sum float64
    for _, num := range countArray {
        sum += float64(num[0])
    }
    creditArray := calculateCredit(countArray)
    for i := 0; i < len(antiCheatIds); i++ {
        accountAsBytes, err := stub.GetState(antiCheatIds[i])
        if err != nil {
            return err
        }
        var account Account
        err = json.Unmarshal(accountAsBytes, &account)
        if err != nil {
            return err
        }
        assets, err := strconv.ParseFloat(account.Assets, 64)
        assets += float64(countArray[i][0]) / sum
        account.Assets = strconv.FormatFloat(assets, 'E', -1, 64)
        credit, err := strconv.ParseFloat(account.Credit, 64)
        credit += creditArray[i]
        account.Credit = strconv.FormatFloat(credit, 'E', -1, 64)
        accountAsBytes, _ = json.Marshal(account)
        stub.PutState(antiCheatIds[i], accountAsBytes)
    }
    return nil
}

func calculateCredit(countArray [][2]int) []float64 {
    var length = len(countArray)
    var pointArray = make([]float64, length)
    var sum float64
    for i := 0; i < length; i++ {
        pointArray[i] = float64(countArray[i][0]*RIGHT_CREDIT - countArray[i][1]*WRONG_CREDIT)
        sum += pointArray[i]
    }
    avg := sum / float64(length)
    for i := 0; i < length; i++ {
        pointArray[i] -= avg
    }
    return pointArray
}

// main function starts up the chaincode in the container during instantiate
func main() {
    if err := shim.Start(new(SimpleAsset)); err != nil {
        fmt.Printf("Error starting SimpleAsset chaincode: %s", err)
    }
}
