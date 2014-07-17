package main

import (
	"crypto/rand"
	_ "crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"strings"
)

type Query struct {
	XMLName  xml.Name `xml:"query"`
	Xmlns    string   `xml:"xmlns,attr"`
	Username string   `xml:"username"`
	Password string   `xml:"password"`
	Digest   string   `xml:"digest"`
	Resource string   `xml:"resource"`
}

type Bind struct {
	XMLName  xml.Name `xml:"bind"`
	Xmlns    string   `xml:"xmlns,attr"`
	Resource string   `xml:"resource"`
	JID      string   `xml:"jid"`
}

type Iq struct {
	XMLName xml.Name  `xml:"iq"`
	Type    string    `xml:"type,attr"`
	Id      string    `xml:"id,attr"`
	Query   Query     `xml:"query"`
	Bind    Bind      `xml:"bind,omitempty"`
	Session *struct{} `xml:"session"`
}

type Stream struct {
	XMLName     xml.Name `xml:"stream:stream"`
	From        string   `xml:"from,attr"`
	To          string   `xml:"to,attr"`
	Id          string   `xml:"id,attr"`
	Version     string   `xml:"version,attr"`
	XmlLang     string   `xml:"xml:lang,attr"`
	Xmlns       string   `xml:"xmlns,attr"`
	XmlnsStream string   `xml:"xmlns:stream,attr"`
	Iq          Iq       `xml:"iq"`
}

type Mechanisms struct {
	XMLName      xml.Name `xml:"mechanisms"`
	MechNS       string   `xml:"xmlns,attr"`
	Mechanisms   []string `xml:"mechanisms>mechanism"`
	MechRequired string   `xml:"mechanisms>required"`
}

type Features struct {
	XMLName    xml.Name   `xml:"stream:features"`
	StartTLS   string     `xml:"starttls>optional"`
	Mechanisms Mechanisms `xml:"mechanisms"`
}

func Serve(addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		// handle error
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			// handle error
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {

	b := xml.NewDecoder(conn)

	var element xml.StartElement
	var t xml.Token

	m := &Mechanisms{Mechanisms: []string{"PLAIN"}, MechNS: "urn:ietf:params:xml:ns:xmpp-sasl"}
	f := &Features{Mechanisms: *m}
	features, _ := xml.Marshal(f)
	xmlFeatures := strings.Replace(string(features), "<required></required>", "<required/>", -1)
	xmlFeatures = strings.Replace(xmlFeatures, "<optional></optional>", "<optional/>", -1)

	fmt.Fprintf(conn, `<?xml version='1.0'?>
		<stream:stream
			from='localhost'
			id=`+`'abc123' `+`
			to='james@localhost'
			version='1.0'
			xml:lang='en'
			xmlns='jabber:server'
			xmlns:stream='http://etherx.jabber.org/streams'>`)

	fmt.Println(xmlFeatures)
	fmt.Fprintf(conn, xmlFeatures)
	for {
// 		iqData := new(Iq)
// 		b.Decode(iqData)

		t, _ = b.Token()
		switch t := t.(type) {
		case xml.StartElement:
			element = t

			fmt.Println("ELEMENT: ", element.Name.Local)

			switch element.Name.Local {

			case "auth":
				fmt.Println("AUTHENTICATING:  Granting access")
				fmt.Fprintf(conn, "<success xmlns=\"urn:ietf:params:xml:ns:xmpp-sasl\"/>")
				fmt.Fprintf(conn, `<?xml version='1.0'?>
							<stream:stream
								from='localhost'
								id=`+`'abc321' `+`
								version='1.0'
								xml:lang='en'
								xmlns='jabber:client'
								xmlns:stream='http://etherx.jabber.org/streams'>
					<stream:features>
					<bind xmlns="urn:ietf:params:xml:ns:xmpp-bind">
					<required/>
					</bind>
					<session xmlns="urn:ietf:params:xml:ns:xmpp-session">
					<optional/>
					</session>
					</stream:features>`)

			case "resource":
			  	iqData := new(Iq)
				b.DecodeElement(&iqData, nil)
				fmt.Println("IQ Data type: ", iqData.Type)
				fmt.Fprintf(conn, `<iq id="%s" type="result"><bind xmlns="urn:ietf:params:xml:ns:xmpp-bind"><jid>james@localhost/tesla</jid></bind>`, iqData.Id)

			case "iq":
				fmt.Println("IN iq")
				iqData := new(Iq)
				b.DecodeElement(&iqData, &element)
				fmt.Println("IQ Data type: ", iqData.Type)
				switch iqData.Type {
				case "get":
					r := &Iq{Id: iqData.Id, Type: "result"}
					r.Query = Query{Xmlns: "jabber:iq:auth"}
					output, _ := xml.Marshal(r)
					fmt.Fprintf(conn, string(output))
				case "set":
					// Need to perform auth lookup here
					fmt.Println("Got a set request")
				  fmt.Println("Bind resource: ", iqData.Bind.Resource)
					if iqData.Bind.Resource != "" {
						fmt.Fprintf(conn, `<iq id="%s" type="result"><bind xmlns="urn:ietf:params:xml:ns:xmpp-bind"><jid>james@localhost/tesla</jid></bind></iq>`, iqData.Id)
					} else {
						i := &Iq{Id: iqData.Id, Type: "result"}
						output, _ := xml.Marshal(i)
						fmt.Fprint(conn, string(output))

					}

				}

			}

		}

	}

}

func getId() string {
	b := make([]byte, 8)
	io.ReadFull(rand.Reader, b)
	return fmt.Sprintf("%x", b)
}
