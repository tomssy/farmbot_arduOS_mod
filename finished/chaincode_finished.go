package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

var FarmWeatherIndexStr = "_farmindex"    //name for the key/value that will store a list of all known marbles
var ActiveInsuranceStr = "_openinsurance" //name for the key/value that will store all open trades
var UserIndexStr = "_userindex"

type Weather struct {
	Name        string `json:"name"`        // rainy sunny cloudy
	Temperature int    `json:"temperature"` // -274 C - max int
}

type Farm struct {
	Name         string    `json:"name"` //the fieldtags are needed to keep case from bouncing around
	Address      string    `json:"address"`
	Owner        string    `json:"owner"`
	WeatherIndex []Weather `json:"weather_index"`
}

type User struct {
	Name string `json:"name"`
	Coin int    `json:"Coin"`
}

type AnInsurance struct { //when bad things happen the beneficiaries get coin = Number * Rate
	Insurant      string `json:"insurant"`      // who is the target we will protect farm name
	Beneficiaries string `json:"beneficiaries"` // who will beneficial from this insurance user name
	Timestamp     int64  `json:"timestamp"`     // when this insurance entry into force
	Number        int    `json:"number"`        // Number of insured
	Rate          int    `json:"rate"`          // decide how many coins beneficiaries will get.
	State         string `json:"state"`         // wait active end
}

type ActiveInsurance struct {
	AllInsurance []AnInsurance `json:"all_insurance"`
}

// ============================================================================================================================
// Main
// ============================================================================================================================
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

// ============================================================================================================================
// Init - reset all the things
// ============================================================================================================================
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	var Aval int
	var err error

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	// Initialize the chaincode
	Aval, err = strconv.Atoi(args[0])
	if err != nil {
		return nil, errors.New("Expecting integer value for asset holding")
	}

	// Write the state to the ledger
	err = stub.PutState("abc", []byte(strconv.Itoa(Aval))) //making a test var "abc", I find it handy to read/write to it right away to test the network
	if err != nil {
		return nil, err
	}

	var empty []string
	jsonAsBytes, _ := json.Marshal(empty) //marshal an emtpy array of strings to clear the index
	err = stub.PutState(FarmWeatherIndexStr, jsonAsBytes)
	if err != nil {
		return nil, err
	}

	err = stub.PutState(UserIndexStr, jsonAsBytes)
	if err != nil {
		return nil, err
	}

	var insurances ActiveInsurance
	jsonAsBytes, _ = json.Marshal(insurances) //clear the open trade struct
	err = stub.PutState(ActiveInsuranceStr, jsonAsBytes)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// ============================================================================================================================
// Run - Our entry point for Invocations - [LEGACY] obc-peer 4/25/2016
// ============================================================================================================================
func (t *SimpleChaincode) Run(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("run is running " + function)
	return t.Invoke(stub, function, args)
}

// ============================================================================================================================
// Invoke - Our entry point for Invocations
// ============================================================================================================================
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "init" { //initialize the chaincode state, used as reset
		return t.Init(stub, "init", args)
	} else if function == "write" { //writes a value to the chaincode state
		return t.Write(stub, args)
	} else if function == "create_user" { //create a new marble
		return t.create_user(stub, args)
	} else if function == "create_farm" { //create a new trade order
		return t.create_farm(stub, args)
	} else if function == "create_insurance" { //forfill an open trade order
		return t.create_insurance(stub, args)
		fmt.Println("create_insurance")
	} else if function == "update_weather" { //cancel an open trade order
		return t.update_weather(stub, args)
		fmt.Println("update weather")
	}
	fmt.Println("invoke did not find func: " + function) //error

	return nil, errors.New("Received unknown function invocation")
}

// ============================================================================================================================
// Query - Our entry point for Queries
// ============================================================================================================================
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("query is running " + function)

	// Handle different functions
	if function == "read" { //read a variable
		return t.read(stub, args)
	}
	fmt.Println("query did not find func: " + function) //error

	return nil, errors.New("Received unknown function query")
}

// ============================================================================================================================
// Read - read a variable from chaincode state
// ============================================================================================================================
func (t *SimpleChaincode) read(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var name, jsonResp string
	var err error

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting name of the var to query")
	}

	name = args[0]
	valAsbytes, err := stub.GetState(name) //get the var from chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + name + "\"}"
		return nil, errors.New(jsonResp)
	}

	return valAsbytes, nil //send it onward
}

// ============================================================================================================================
// Write - write variable into chaincode state
// ============================================================================================================================
func (t *SimpleChaincode) Write(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var name, value string // Entities
	var err error
	fmt.Println("running write()")

	if len(args) != 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 2. name of the variable and value to set")
	}

	name = args[0] //rename for funsies
	value = args[1]
	err = stub.PutState(name, []byte(value)) //write the variable into the chaincode state
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// ============================================================================================================================
// Create User - create a new User,
// ============================================================================================================================

func (t *SimpleChaincode) create_user(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var err error

	//   0       1       2     3
	//  'name'   'money'
	if len(args) != 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 4")
	}

	//input sanitation
	fmt.Println("- start create user")
	if len(args[0]) <= 0 {
		return nil, errors.New("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return nil, errors.New("2nd argument must be a non-empty string")
	}

	name := strings.ToLower(args[0])
	coin, err := strconv.Atoi(args[1])
	if err != nil {
		return nil, errors.New("2rd argument must be a numeric string")
	}

	//check if marble already exists
	UserAsBytes, err := stub.GetState(name)
	if err != nil {
		return nil, errors.New("Failed to get marble name")
	}
	res := User{}
	json.Unmarshal(UserAsBytes, &res)
	if res.Name == name {
		fmt.Println("This user arleady exists: " + name)
		fmt.Println(res)
		return nil, errors.New("This user arleady exists") //all stop a user by this name exists
	}

	//build the user json string manually
	var user User
	user.Name = name
	user.Coin = coin
	UserAsBytes, err = json.Marshal(user)
	err = stub.PutState(name, UserAsBytes) //store marble with id as key
	if err != nil {
		return nil, err
	}

	//get the marble index
	UsersAsBytes, err := stub.GetState(UserIndexStr)
	if err != nil {
		return nil, errors.New("Failed to get marble index")
	}
	var UserIndex []string
	json.Unmarshal(UsersAsBytes, &UserIndex) //un stringify it aka JSON.parse()

	//append
	UserIndex = append(UserIndex, name) //add marble name to index list
	fmt.Println("! User index: ", UserIndex)
	jsonAsBytes, _ := json.Marshal(UserIndex)
	err = stub.PutState(UserIndexStr, jsonAsBytes) //store name of marble

	fmt.Println("- end create User")
	return nil, nil
}

// ============================================================================================================================
// create farm
// ============================================================================================================================
func (t *SimpleChaincode) create_farm(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var err error

	//   0       1       2     3                4
	//  'name'   'addre' 'own'  'weathername'  Temperature
	if len(args) <= 4 {
		return nil, errors.New("Incorrect number of arguments. Expecting >=4")
	}
	stub.PutState("start create farm", []byte(strings.ToLower(args[0])))
	//input sanitation
	fmt.Println("- start create farm")
	if len(args[0]) <= 0 {
		return nil, errors.New("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return nil, errors.New("2nd argument must be a non-empty string")
	}
	if len(args[2]) <= 0 {
		return nil, errors.New("2nd argument must be a non-empty string")
	}
	if len(args[3]) <= 0 {
		return nil, errors.New("2nd argument must be a non-empty string")
	}
	newfarm := Farm{}
	name := strings.ToLower(args[0])
	newfarm.Name = name
	newfarm.Address = strings.ToLower(args[1])
	newfarm.Owner = strings.ToLower(args[2])

	fmt.Println("- create new farm")
	jsonAsBytes, _ := json.Marshal(newfarm)
	err = stub.PutState("_debug1", jsonAsBytes)

	for i := 3; i < len(args); i++ { //create and append each willing trade
		Temperature, err := strconv.Atoi(args[i+1])
		if err != nil {
			msg := "is not a numeric string " + args[i+1]
			fmt.Println(msg)
			return nil, errors.New(msg)
		}

		Weather_now := Weather{}
		Weather_now.Name = args[i]
		Weather_now.Temperature = Temperature
		fmt.Println("! created weather: " + args[i])
		jsonAsBytes, _ = json.Marshal(Weather_now)
		err = stub.PutState("_debug2", jsonAsBytes)

		newfarm.WeatherIndex = append(newfarm.WeatherIndex, Weather_now)
		fmt.Println("! appended weather")
		i++
	}

	//check if farm already exists
	FarmAsBytes, err := stub.GetState(name)
	if err != nil {
		return nil, errors.New("Failed to get farm name")
	}

	res := Farm{}
	json.Unmarshal(FarmAsBytes, &res)
	if res.Name == name {
		fmt.Println("This farm arleady exists: " + name)
		fmt.Println(res)
		return nil, errors.New("This farm arleady exists") //all stop a user by this name exists
	}

	newfarmAsBytes, _ := json.Marshal(newfarm)
	stub.PutState(name, newfarmAsBytes)
	//get the marble index
	FarmAsBytes, err = stub.GetState(FarmWeatherIndexStr)
	if err != nil {
		return nil, errors.New("Failed to get marble index")
	}
	var FarmIndex []string

	json.Unmarshal(FarmAsBytes, &FarmIndex) //un stringify it aka JSON.parse()
	//append
	FarmIndex = append(FarmIndex, name) //add marble name to index list
	FarmAsBytes, _ = json.Marshal(FarmIndex)
	err = stub.PutState(FarmWeatherIndexStr, FarmAsBytes) //store name of marble

	fmt.Println("- end create User")
	return nil, nil
}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

// ============================================================================================================================
// Create User - create a new User,
// ============================================================================================================================

func (t *SimpleChaincode) create_insurance(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var err error

	//   0       1       2     3
	//  'insurant'   'beneficial' 'Number' 'rate' 'state'
	if len(args) != 5 {
		return nil, errors.New("Incorrect number of arguments. Expecting 5")
	}

	//input sanitation
	fmt.Println("- start create user")
	if len(args[0]) <= 0 {
		return nil, errors.New("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return nil, errors.New("2nd argument must be a non-empty string")
	}
	if len(args[2]) <= 0 {
		return nil, errors.New("2nd argument must be a non-empty string")
	}
	if len(args[3]) <= 0 {
		return nil, errors.New("2nd argument must be a non-empty string")
	}
	if len(args[4]) <= 0 {
		return nil, errors.New("2nd argument must be a non-empty string")
	}

	new_insurance := AnInsurance{}
	new_insurance.Insurant = strings.ToLower(args[0])
	new_insurance.Beneficiaries = strings.ToLower(args[1])
	new_insurance.Number, err = strconv.Atoi(args[2])
	if err != nil {
		return nil, errors.New("3rd argument must be a numeric string")
	}
	new_insurance.Rate, err = strconv.Atoi(args[3])
	if err != nil {
		return nil, errors.New("4rd argument must be a numeric string")
	}
	new_insurance.State = strings.ToLower(args[4])
	new_insurance.Timestamp = makeTimestamp()

	jsonAsBytes, _ := json.Marshal(new_insurance)
	err = stub.PutState("_debug1", jsonAsBytes)

	//get the marble index
	InsuranceAsBytes, err := stub.GetState(ActiveInsuranceStr)
	if err != nil {
		return nil, errors.New("Failed to get marble index")
	}
	var Insurances ActiveInsurance
	json.Unmarshal(InsuranceAsBytes, &Insurances) //un stringify it aka JSON.parse()

	//append
	Insurances.AllInsurance = append(Insurances.AllInsurance, new_insurance) //add marble name to index list
	InsuranceAsBytes, _ = json.Marshal(Insurances)
	err = stub.PutState(ActiveInsuranceStr, InsuranceAsBytes)

	fmt.Println("- end create User")
	return nil, nil
}

func (t *SimpleChaincode) update_weather(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var err error

	//   0       1       2     3
	//  'farm_name'   'weather type' 'Temperature'
	if len(args) != 3 {
		return nil, errors.New("Incorrect number of arguments. Expecting 5")
	}

	//input sanitation
	fmt.Println("- start create user")
	if len(args[0]) <= 0 {
		return nil, errors.New("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return nil, errors.New("2nd argument must be a non-empty string")
	}
	if len(args[2]) <= 0 {
		return nil, errors.New("2nd argument must be a non-empty string")
	}

	Temperature, err := strconv.Atoi(args[2])
	if err != nil {
		msg := "is not a numeric string " + args[2]
		fmt.Println(msg)
		return nil, errors.New(msg)
	}
	Weather_now := Weather{}
	Weather_now.Name = args[1]
	Weather_now.Temperature = Temperature

	jsonAsBytes, _ := json.Marshal(Weather_now)
	err = stub.PutState("_debug2", jsonAsBytes)
	farmname := strings.ToLower(args[0])
	farmAsByte, err := stub.GetState(farmname)
	if err != nil {
		return nil, errors.New("farm not exist")
	}

	var update_farm Farm
	json.Unmarshal(farmAsByte, &update_farm)

	jsonAsBytes, _ = json.Marshal(update_farm.WeatherIndex)

	err = stub.PutState("_debug4", jsonAsBytes)

	update_farm.WeatherIndex = append(update_farm.WeatherIndex, Weather_now)

	farmAsByte, err = json.Marshal(update_farm)
	if err != nil {
		return nil, errors.New("farm marshal fail")
	}
	stub.PutState(farmname, farmAsByte)

	err = stub.PutState("_debug3", farmAsByte)

	//check if terrible weather
	if len(update_farm.WeatherIndex) >= 3 {
		var Insurances ActiveInsurance
		InsuranceAsBytes, err := stub.GetState(ActiveInsuranceStr)
		if err != nil {
			return nil, errors.New("insurance get error")
		}
		json.Unmarshal(InsuranceAsBytes, &Insurances) //un stringify it aka JSON.parse()
		bad_count := 0
		wl := len(update_farm.WeatherIndex)
		for i := wl - 3; i < wl; i++ {
			Weather := update_farm.WeatherIndex[i]
			if Weather.Name == "rainy" {
				bad_count += 1
			}
			if Weather.Name == "sunny" {
				bad_count = 0
			}
		}
		if bad_count >= 3 {
			for i, val := range Insurances.AllInsurance {
				if val.State == "actived" && val.Insurant == farmname {
					Insurances.AllInsurance[i].State = "solved"
					benefit := val.Number * val.Rate
					username := val.Beneficiaries
					var lucky_dog User
					userAsByte, err := stub.GetState(username)
					if err != nil {
						return nil, errors.New("user don't exist")
					}
					json.Unmarshal(userAsByte, &lucky_dog)
					lucky_dog.Coin = lucky_dog.Coin + benefit
					userAsByte, err = json.Marshal(lucky_dog)
					stub.PutState(username, userAsByte)

				}
			}
		}

		InsuranceAsBytes, err = json.Marshal(Insurances)
		stub.PutState(ActiveInsuranceStr, InsuranceAsBytes)
	}

	return nil, nil
}
