package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Juniper/go-netconf/netconf"
	xj "github.com/basgys/goxml2json"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"strings"
	"time"
)

var (
	host     = flag.String("host", "localhost", "Hostname")
	username = flag.String("username", "netconf", "Username")
	passwd   = flag.String("password", "netconf", "Password")
	source   = flag.String("source", "running", "Source datastore")
	target   = flag.String("target", "running", "Target datastore")
	subtree  = flag.String("subtree", "netconf-server", "Subtree or module to select")
	file     = flag.String("file", "", "Configuration file")
	action   = flag.String("action", "get", "Netconf command: get or edit")
	timeout  = flag.Int("timeout", 30, "Timeout")

	getConfigXml  = "<get-config><source><%s/></source><filter type=\"subtree\"><%s/></filter></get-config>"
	editConfigXml = "<edit-config><target><%s/></target><config>%s</config></edit-config>"
)

func main() {
	if flag.Parse(); flag.Parsed() == false {
		log.Fatal("Syntax error!")
		return
	}

	switch *action {
	case "get":
		getConfig()
	case "edit":
		editConfig()
	}
}

func getConfig() {
	session := startSSHSession()
	if session == nil {
		return
	}
	defer session.Close()

	cmd := netconf.RawMethod(fmt.Sprintf(getConfigXml, *source, *subtree))
	reply, err := session.Exec(cmd)
	if err != nil {
		log.Fatal(err)
		return
	}
	displayReply(reply.RawReply)
}

func editConfig() {
	if *file == "" {
		log.Fatal("Configuration file missing!")
		return
	}

	session := startSSHSession()
	if session == nil {
		return
	}
	defer session.Close()

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
