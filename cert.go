package api

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"net"
	"strings"

	"git.quyun.com/apibox/pki"
)

// 生成指定域名列表和IP的证书
// 多个域名或IP有,分隔
func makeCert(domains, ip string) (certPemBlock, keyPemBlock []byte, err error) {
	dnsNames := []string{}
	ipAddresses := []net.IP{}

	if domains != "" {
		ds := strings.Split(domains, ",")
		// 检测dnsNames里是否有IP地址，否则连接时可能出现以下错误：
		// x509: cannot validate certificate for 117.121.10.79 because it doesn't contain any IP SANs
		for _, d := range ds {
			if ip := net.ParseIP(d); ip != nil {
				ipAddresses = append(ipAddresses, ip)
			} else {
				dnsNames = append(dnsNames, d)
			}
		}
	}
	if ip != "*" && ip != "0.0.0.0" && ip != "" {
		ipAddresses = append(ipAddresses, net.ParseIP(ip))
	}
	if len(dnsNames) == 0 && len(ipAddresses) == 0 {
		// 如果都没有设置，则取所有网卡IP
		// outingIp := getOutgoingIp()
		// if outingIp != "" {
		// 	ipAddresses = []net.IP{net.ParseIP(outingIp)}
		// }
		ifAddrs, err := net.InterfaceAddrs()
		if err != nil {
			return nil, nil, err
		}
		for _, ifAddr := range ifAddrs {
			if ifAddr.Network() == "ip+net" {
				ifAddrStr := ifAddr.String()
				ifAddrSepPos := strings.IndexByte(ifAddrStr, '/')
				if ifAddrSepPos <= 0 {
					continue
				}
				ipAddr := ifAddrStr[:ifAddrSepPos]
				ipAddresses = append(ipAddresses, net.ParseIP(ipAddr))
			}
		}
	}

	// 如果未绑定域名，则 Common Name 设置为IP，否则 Common Name 为第一个域名
	var commonName string
	if len(dnsNames) > 0 {
		commonName = dnsNames[0]
	} else if len(ipAddresses) > 0 {
		commonName = ipAddresses[0].String()
	}

	certBytes, privKey, err := pki.CreateX509Cert(pki.RootCert, pki.RootKey, commonName, dnsNames, ipAddresses)

	var pemcert = &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}
	certPemBlock = pem.EncodeToMemory(pemcert)

	var pemkey = &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	}
	keyPemBlock = pem.EncodeToMemory(pemkey)

	return
}

// // 获得出口网卡的ip 地址
// func getOutgoingIp() string {
// 	conn, err := net.Dial("udp", "114.114.114.114:53")
// 	if err != nil {
// 		return ""
// 	}

// 	localAddr := conn.LocalAddr().String()
// 	return localAddr[:strings.IndexByte(localAddr, ':')]
// }

func loadX509PemBlock(certFile, keyFile string) (certPemBlock, keyPemBlock []byte, err error) {
	certPemBlock, err = ioutil.ReadFile(certFile)
	if err != nil {
		return
	}
	if p, _ := pem.Decode(certPemBlock); p == nil {
		err = errors.New("not pem format of file: " + certFile)
		return
	}
	keyPemBlock, err = ioutil.ReadFile(keyFile)
	if err != nil {
		return
	}
	if p, _ := pem.Decode(certPemBlock); p == nil {
		err = errors.New("not pem format of file: " + keyFile)
		return
	}
	return
}
