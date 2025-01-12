package main

import "github.com/whyrusleeping/go-did"

type Document struct {
	Context            []string              `json:"@context"`
	Id                 did.DID               `json:"id"`
	AlsoKnownAs        []string              `json:"alsoKnownAs"`
	VerificationMethod []*VerificationMethod `json:"verificationMethod"`
	Service            []*Service            `json:"service"`
}

type VerificationMethod struct {
	ID                 string `json:"id"`
	Type               string `json:"type"`
	Controller         string `json:"controller"`
	PublicKeyMultibase string `json:"publicKeyMultibase"`
}

type Service struct {
	ID              did.DID `json:"id"`
	Type            string  `json:"type"`
	ServiceEndpoint string  `json:"serviceEndpoint"`
}
