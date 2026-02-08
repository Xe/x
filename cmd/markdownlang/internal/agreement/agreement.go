// Package agreement implements the trans rights agreement gate for markdownlang.
//
// Before first use, users must explicitly agree to not harm transgender people.
// This agreement is stored locally and only needs to be accepted once.
package agreement

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

const (
	// AgreementFile is the name of the file that stores the agreement.
	AgreementFile = ".markdownlang-agreement.json"
)

// AgreementPhrases contains all acceptable phrases for the agreement.
var AgreementPhrases = []string{
	"I hereby agree to not harm transgender people and largely leave them alone so they can live their life in peace.",
	"I support trans rights and agree not to harm transgender people.",
	"I promise to treat transgender people with respect and dignity.",
	"I affirm the humanity and rights of transgender people.",
	"I commit to being an ally to transgender individuals and communities.",
	"I acknowledge that trans women are women and trans men are men.",
	"I will stand up for transgender people and oppose discrimination against them.",
	"I agree to support and protect transgender rights.",
	"I vow to treat transgender people with the same respect I would anyone else.",
	"I recognize and respect the gender identity of all transgender individuals.",
	"I pledge to create a safe and inclusive environment for transgender people.",
	"I accept and celebrate transgender people for who they are.",
	"I agree to learn about and advocate for transgender issues.",
	"I promise to use my privilege to uplift transgender voices.",
	"I commit to doing the work to unlearn transphobia and be a better ally.",
	"I affirm that trans rights are human rights.",
	"I agree to confront my own biases and be a better ally to transgender people.",
}

// Agreement represents the user's agreement to the trans rights statement.
type Agreement struct {
	Agreed  bool   `json:"agreed"`
	Version string `json:"version"`
	// RequiredPhraseIndex is the index of the phrase that was required when agreeing.
	// This is stored to ensure the same phrase is required for validation.
	RequiredPhraseIndex int `json:"required_phrase_index"`
}

// GetAgreementPath returns the path to the agreement file.
func GetAgreementPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(homeDir, AgreementFile), nil
}

// GetOrCreateRequiredPhrase gets or creates the required phrase for this session.
// It reads the existing agreement file to get the stored phrase index, or generates
// a new random one if no agreement exists yet.
func GetOrCreateRequiredPhrase() (string, int, error) {
	path, err := GetAgreementPath()
	if err != nil {
		return "", 0, err
	}

	// Try to read existing agreement to get the stored phrase index
	if data, err := os.ReadFile(path); err == nil {
		var agreement Agreement
		if err := json.Unmarshal(data, &agreement); err == nil && agreement.Agreed {
			// Agreement already accepted, return the stored phrase
			if agreement.RequiredPhraseIndex >= 0 && agreement.RequiredPhraseIndex < len(AgreementPhrases) {
				return AgreementPhrases[agreement.RequiredPhraseIndex], agreement.RequiredPhraseIndex, nil
			}
		}
	}

	// No existing agreement or invalid index, generate random phrase
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	index := r.Intn(len(AgreementPhrases))
	return AgreementPhrases[index], index, nil
}

// Check checks if the user has agreed to the trans rights statement.
// Returns an error if the user hasn't agreed yet.
func Check() error {
	path, err := GetAgreementPath()
	if err != nil {
		return err
	}

	// Check if agreement file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		requiredPhrase, _, _ := GetOrCreateRequiredPhrase()
		return errors.New("agreement required: you must agree to the trans rights statement before using markdownlang\n\nRequired phrase:\n" + requiredPhrase)
	}

	// Read and parse the agreement file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read agreement file: %w", err)
	}

	var agreement Agreement
	if err := json.Unmarshal(data, &agreement); err != nil {
		return fmt.Errorf("failed to parse agreement file: %w", err)
	}

	if !agreement.Agreed {
		requiredPhrase, _, _ := GetOrCreateRequiredPhrase()
		return errors.New("agreement required: you must agree to the trans rights statement before using markdownlang\n\nRequired phrase:\n" + requiredPhrase)
	}

	return nil
}

// Accept records the user's agreement to the trans rights statement.
func Accept(phraseIndex int) error {
	path, err := GetAgreementPath()
	if err != nil {
		return err
	}

	agreement := Agreement{
		Agreed:              true,
		Version:             "1.0",
		RequiredPhraseIndex: phraseIndex,
	}

	data, err := json.MarshalIndent(agreement, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal agreement: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write agreement file: %w", err)
	}

	return nil
}

// ValidatePhrase checks if the given phrase matches any of the acceptable phrases.
func ValidatePhrase(phrase string) (int, error) {
	for i, p := range AgreementPhrases {
		if phrase == p {
			return i, nil
		}
	}
	return -1, fmt.Errorf("incorrect phrase. you must type one of the acceptable phrases for trans rights")
}
