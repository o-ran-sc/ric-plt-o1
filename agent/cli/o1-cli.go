package main

import (
	"log"
	"strings"
	"time"
	"os"
	"bytes"
	"flag"
	"fmt"
	"encoding/json"
	"io/ioutil"
	"github.com/Juniper/go-netconf/netconf"
	xj "github.com/basgys/goxml2json"
	"golang.org/x/crypto/ssh"
)

var (
	host     = flag.String("host", "localhost", "Hostname")
	username = flag.String("username", "netconf", "Username")
	passwd   = flag.String("password", "netconf", "Password")
	source   = flag.String("source", "running", "Source datastore")
	target   = flag.String("target", "running", "Target datastore")
	subtree  = flag.String("subtree", "netconf-server", "Subtree or module to select")
	namespace = flag.String("namespace", "urn:o-ran:ric:gnb-status:1.0", "XML namespace")
	file     = flag.String("file", "", "Configuration file")
	action   = flag.String("action", "get", "Netconf command: get or edit")
	timeout  = flag.Int("timeout", 30, "Timeout")

	getStateXml   = "<get><filter type=\"subtree\"><ric xmlns=\"%s\"></ric></filter></get>"
	getConfigXml  = "<get-config><source><%s/></source><filter type=\"subtree\"><%s/></filter></get-config>"
	editConfigXml = "<edit-config><target><%s/></target><config>%s</config></edit-config>"
)

func main() {
	defer func() { // catch or finally
        if err := recover(); err != nil { //catch
            fmt.Fprintf(os.Stderr, "Something went wrong: %v\n", err)
            os.Exit(1)
        }
	}()

	if flag.Parse(); flag.Parsed() == false {
		log.Fatal("Syntax error!")
		return
	}

	session := startSSHSession()
	if session == nil {
		return
	}
	defer session.Close()

	switch *action {
	case "get":
		getStatus(session, getStateXml)
	case "get-config":
		getConfig(session, getConfigXml)
	case "edit":
		editConfig(session)
	}
}

func getStatus(session *netconf.Session, cmdXml string) {
	cmd := netconf.RawMethod(fmt.Sprintf(cmdXml, *namespace))
	reply, err := session.Exec(cmd)
	if err != nil {
		log.Fatal(err)
		return
	}
	displayReply(reply.RawReply)
}

func getConfig(session *netconf.Session, cmdXml string) {
	cmd := netconf.RawMethod(fmt.Sprintf(cmdXml, *source, *subtree))
	reply, err := session.Exec(cmd)
	if err != nil {
		log.Fatal(err)
		return
	}
	displayReply(reply.RawReply)
}

func editConfig(session *netconf.Session) {
	if *file == "" {
		log.Fatal("Configuration file missing!")
		return
	}

	if data, err := ioutil.ReadFile(*file); err == nil {
		cmd := netconf.RawMethod(fmt.Sprintf(editConfigXml, *target, data))
		reply, err := session.Exec(cmd)
		if err != nil {
			log.Fatal(err)
			return
		}
		displayReply(reply.RawReply)
	}
}

func startSSHSession() *netconf.Session {
	sshConfig := &ssh.ClientConfig{
		User:            *username,
		Auth:            []ssh.AuthMethod{ssh.Password(*passwd)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(*timeout) * time.Second,
	}

	session, err := netconf.DialSSH(*host, sshConfig)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	return session
}

func prettyPrint(b string) string {
	var out bytes.Buffer
	if err := json.Indent(&out, []byte(b), "", "  "); err == nil {
		return string(out.Bytes())
	}
	return ""
}

func displayReply(rawReply string) {
	xml := strings.NewReader(rawReply)
	json, err := xj.Convert(xml)
	if err != nil {
		log.Fatal("Something went sore ... XML is invalid!")
	}
	fmt.Println(prettyPrint(json.String()))
}
