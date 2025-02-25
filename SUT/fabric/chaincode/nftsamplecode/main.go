/*
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"fmt"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/hyperledger/fabric-contract-api-go/metadata"
	"github.com/msalimbene/hlp-721/etcdv3"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

type serverConfig struct {
	CCID    string
	Address string
}

func main() {
	hlpNftContract := new(TokenERC721Contract)
	hlpNftContract.Info.Version = "0.0.1"
	hlpNftContract.Info.Description = "ERC-721 fabric port"
	hlpNftContract.Info.License = new(metadata.LicenseMetadata)
	hlpNftContract.Info.License.Name = "Apache-2.0"
	hlpNftContract.Info.Contact = new(metadata.ContactMetadata)
	hlpNftContract.Info.Contact.Name = "Matias Salimbene"

	chaincode, err := contractapi.NewChaincode(hlpNftContract)
	chaincode.Info.Title = "ERC-721 chaincode"
	chaincode.Info.Version = "0.0.1"

	if err != nil {
		panic("Could not create chaincode from TokenERC721Contract." + err.Error())
	}

	etcdAddress := os.Getenv("ETCDADDRESS")
	podIp := os.Getenv("POD_IP")
	podPort := os.Getenv("POD_PORT")
	chaincodeAdress := podIp + podPort

	config := serverConfig{
		CCID:    os.Getenv("CHAINCODE_ID"),
		Address: chaincodeAdress,
	}

	// 定义 etcd 地址
	var etcdEndpoints = []string{
		etcdAddress,
	}
	// 把服务注册到 etcd
	etcdServ, err := etcdv3.NewServiceRegister(
		etcdEndpoints,
		os.Getenv("CHAINCODE_CCID")+"+"+chaincodeAdress,
		chaincodeAdress,
		5)

	if err != nil {
		log.Fatalf("服务注册出错: %v", err)
	}
	defer etcdServ.Close()

	c := make(chan os.Signal)
	// 监听信号
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		for s := range c {
			switch s {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM:
				fmt.Println("退出:", s)
				etcdServ.Close()
				os.Exit(0)
			default:
				fmt.Println("其他信号:", s)
			}
		}
	}()

	server := &shim.ChaincodeServer{
		CCID:     config.CCID,
		Address:  config.Address,
		CC:       chaincode,
		TLSProps: getTLSProperties(),
	}

	if err := server.Start(); err != nil {
		log.Panicf("error starting asset-transfer-basic chaincode: %s", err)
	}
}

func getEnvOrDefault(env, defaultVal string) string {
	value, ok := os.LookupEnv(env)
	if !ok {
		value = defaultVal
	}
	return value
}

// Note that the method returns default value if the string
// cannot be parsed!
func getBoolOrDefault(value string, defaultVal bool) bool {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultVal
	}
	return parsed
}

func getTLSProperties() shim.TLSProperties {
	// Check if chaincode is TLS enabled
	tlsDisabledStr := getEnvOrDefault("CHAINCODE_TLS_DISABLED", "true")
	key := getEnvOrDefault("CHAINCODE_TLS_KEY", "")
	cert := getEnvOrDefault("CHAINCODE_TLS_CERT", "")
	clientCACert := getEnvOrDefault("CHAINCODE_CLIENT_CA_CERT", "")

	// convert tlsDisabledStr to boolean
	tlsDisabled := getBoolOrDefault(tlsDisabledStr, false)
	var keyBytes, certBytes, clientCACertBytes []byte
	var err error

	if !tlsDisabled {
		keyBytes, err = ioutil.ReadFile(key)
		if err != nil {
			log.Panicf("error while reading the crypto file: %s", err)
		}
		certBytes, err = ioutil.ReadFile(cert)
		if err != nil {
			log.Panicf("error while reading the crypto file: %s", err)
		}
	}
	// Did not request for the peer cert verification
	if clientCACert != "" {
		clientCACertBytes, err = ioutil.ReadFile(clientCACert)
		if err != nil {
			log.Panicf("error while reading the crypto file: %s", err)
		}
	}

	return shim.TLSProperties{
		Disabled:      tlsDisabled,
		Key:           keyBytes,
		Cert:          certBytes,
		ClientCACerts: clientCACertBytes,
	}
}
