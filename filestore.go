// internal/datastore/filestore.go
package datastore

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	// Adjust import paths based on your go.mod module name
	"github.com/yackko/satcom-code/internal/config"
	"github.com/yackko/satcom-code/internal/crypto" // Ensure this path is correct
	"github.com/yackko/satcom-code/types"

	"golang.org/x/term"
)

var (
	satellitesData     = make(map[string]types.Satellite) // Renamed to avoid conflict if types.Satellite was just Satellite
	dataFileLock       sync.Mutex
	dataPath           string
	passphraseProvided bool   // Indicates if a valid passphrase was used to unlock/init
	sessionKey         []byte // The key derived from the passphrase for the current session
)

// getPassphrase securely gets the passphrase, preferring env var, then prompting.
func getPassphrase(promptForCreation bool) (string, error) {
	passphrase := os.Getenv(config.PassphraseEnvVar)
	if passphrase != "" {
		return passphrase, nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("%s environment variable not set and not running in a terminal to prompt for passphrase", config.PassphraseEnvVar)
	}
	fmt.Fprint(os.Stderr, "Enter passphrase for datastore: ")
	bytePassphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr) // Newline after input
	if err != nil {
		return "", fmt.Errorf("failed to read passphrase: %w", err)
	}
	passphrase = string(bytePassphrase)
	if promptForCreation && passphrase != "" {
		fmt.Fprint(os.Stderr, "Confirm passphrase: ")
		bytePassphraseConfirm, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", fmt.Errorf("failed to read passphrase confirmation: %w", err)
		}
		if passphrase != string(bytePassphraseConfirm) {
			return "", fmt.Errorf("passphrases do not match")
		}
	}
	return passphrase, nil
}

// Init initializes the datastore path and attempts to load data.
func Init() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	exeDir := filepath.Dir(exePath)
	dataPath = filepath.Join(exeDir, config.DataFileName)

	err = load() // load will handle passphrase and decryption
	if err != nil {
		// Check for specific, non-fatal errors related to passphrase or file not existing
		// These allow the CLI to start for commands that don't need datastore access (like 'help' or 'explain')
		if os.IsNotExist(err) {
			// File doesn't exist: this is fine for initial load, passphrase will be requested on first save.
			// load() function might have already printed a notice.
			passphraseProvided = false // No data loaded, no key derived yet
			sessionKey = nil
			return nil
		}
		if strings.Contains(err.Error(), "passphrase") || strings.Contains(err.Error(), "decrypt") {
			fmt.Fprintf(os.Stderr, "Warning: Could not unlock datastore: %v\n", err)
			passphraseProvided = false // Mark as not unlocked
			sessionKey = nil
			return nil // Allow CLI to proceed for non-data commands
		}
		// For other critical errors (e.g., permission issues other than file not existing)
		return err
	}
	return nil
}

// IsUnlocked returns true if the datastore is considered unlocked.
func IsUnlocked() bool {
	return passphraseProvided && len(sessionKey) > 0
}

// GetSatellites returns a copy of all satellite data.
func GetSatellites() (map[string]types.Satellite, error) {
	if !IsUnlocked() {
		return nil, fmt.Errorf("datastore is locked or not initialized. Please set %s or provide passphrase.", config.PassphraseEnvVar)
	}
	dataFileLock.Lock()
	defer dataFileLock.Unlock()
	// Return a copy to prevent external modification
	satsCopy := make(map[string]types.Satellite, len(satellitesData))
	for k, v := range satellitesData {
		satsCopy[k] = v
	}
	return satsCopy, nil
}

// AddSatellite adds or updates a satellite in the in-memory store.
// Save() must be called to persist.
func AddSatellite(sat types.Satellite) error {
	if !IsUnlocked() {
		return fmt.Errorf("datastore is locked. Cannot add/update satellite.")
	}
	dataFileLock.Lock()
	defer dataFileLock.Unlock()
	satellitesData[sat.Name] = sat
	return nil
}

// DeleteSatellite removes a satellite from the in-memory store.
// Save() must be called to persist.
func DeleteSatellite(name string) error {
	if !IsUnlocked() {
		return fmt.Errorf("datastore is locked. Cannot delete satellite.")
	}
	dataFileLock.Lock()
	defer dataFileLock.Unlock()
	if _, exists := satellitesData[name]; !exists {
		return fmt.Errorf("satellite '%s' not found for deletion", name)
	}
	delete(satellitesData, name)
	return nil
}


// load attempts to load and decrypt the datastore.
func load() error {
	// This function is called with dataFileLock already held by Init if restructuring
	// For now, let's assume it needs its own lock or is called carefully.
	// Simpler: init calls this, this handles its own lock.

	_, statErr := os.Stat(dataPath)
	fileExists := !os.IsNotExist(statErr)

	// Try to get passphrase. Prompt for creation (confirmation) only if file does NOT exist.
	currentPassphrase, passErr := getPassphrase(!fileExists)

	if passErr != nil {
		passphraseProvided = false; sessionKey = nil
		if fileExists { return fmt.Errorf("passphrase acquisition failed for existing datastore: %w", passErr) }
		fmt.Fprintf(os.Stderr, "Notice: Datastore file '%s' not found. Passphrase prompt failed or was skipped. First save will require a valid passphrase.\n", dataPath)
		satellitesData = make(map[string]types.Satellite)
		return nil
	}

	if currentPassphrase == "" {
		passphraseProvided = false; sessionKey = nil
		if fileExists { return fmt.Errorf("passphrase not provided for existing datastore '%s'", dataPath) }
		fmt.Fprintf(os.Stderr, "Notice: Datastore file '%s' not found and no passphrase provided. First save will require a valid passphrase.\n", dataPath)
		satellitesData = make(map[string]types.Satellite)
		return nil
	}
	
	passphraseProvided = true // A non-empty passphrase was obtained

	if !fileExists {
		fmt.Fprintf(os.Stderr, "Notice: Datastore file '%s' not found. Will be created and encrypted on first save with the provided passphrase.\n", dataPath)
		satellitesData = make(map[string]types.Satellite)
		// Key will be derived with a new salt during the first save using currentPassphrase
		// We store the passphrase (conceptually, by setting passphraseProvided) but derive key on save with new salt.
		sessionKey = nil // No key derived yet, as no salt from file.
		return nil 
	}

	// File exists, proceed with decryption
	encryptedFileBytes, err := ioutil.ReadFile(dataPath)
	if err != nil {
		passphraseProvided = false; sessionKey = nil
		return fmt.Errorf("failed to read encrypted datastore %s: %w", dataPath, err)
	}

	if len(encryptedFileBytes) < (config.Argon2SaltSize + config.AESGCMNonceSize) {
		passphraseProvided = false; sessionKey = nil
		return fmt.Errorf("encrypted datastore file is too short or corrupted (salt+nonce sections missing)")
	}

	salt := encryptedFileBytes[:config.Argon2SaltSize]
	nonceAndCiphertext := encryptedFileBytes[config.Argon2SaltSize:]

	key, keyErr := crypto.DeriveKeyWithArgon2id(currentPassphrase, salt)
	if keyErr != nil {
		passphraseProvided = false; sessionKey = nil
		return fmt.Errorf("key derivation failed during load: %w", keyErr)
	}
	
	plaintext, err := crypto.Decrypt(nonceAndCiphertext, key)
	if err != nil {
		passphraseProvided = false; sessionKey = nil 
		return err // Decrypt already provides a good error message (passphrase/integrity)
	}

	sessionKey = key // Store derived key for the session if decryption successful

	tempSatellites := make(map[string]types.Satellite)
	if err := json.Unmarshal(plaintext, &tempSatellites); err != nil {
		passphraseProvided = false; sessionKey = nil // Data corrupted after decryption
		return fmt.Errorf("failed to unmarshal decrypted satellite data: %w (data may be corrupt)", err)
	}
	satellitesData = tempSatellites
	if satellitesData == nil { // Should not occur if JSON was valid, even "{}"
		satellitesData = make(map[string]types.Satellite)
	}
	return nil
}

// Save encrypts and writes the current state of satellites.
func Save() error {
	dataFileLock.Lock()
	defer dataFileLock.Unlock()

	if !passphraseProvided {
		// This state implies that InitDataStore/load might have failed to get a passphrase,
		// or the user is trying to save without having unlocked an existing store
		// or without having provided a passphrase for a new store.
		// Re-attempt to get passphrase, this time it's definitively for creation/overwrite.
		fmt.Fprintln(os.Stderr, "Passphrase required to save datastore.")
		currentPassphrase, passErr := getPassphrase(true) // true for confirmation if new
		if passErr != nil || currentPassphrase == "" {
			return fmt.Errorf("passphrase is required to save encrypted datastore: %w (or set %s)", passErr, config.PassphraseEnvVar)
		}
		// If we got here, it means we now have a passphrase.
		passphraseProvided = true // Mark it for the session
		// Key will be derived below with a new salt.
	}

    // Get the passphrase again to ensure we use the latest intended one for this save operation,
    // especially since we will generate a new salt.
    currentPassphrase := os.Getenv(config.PassphraseEnvVar)
    if currentPassphrase == "" { // If not in ENV, it must have been entered via prompt.
        var errPass error
        // We need the raw passphrase. If it was entered via term.ReadPassword, we don't have it anymore.
        // This is a classic key management issue. For now, we re-prompt if not in ENV.
        // This is not ideal UX if they just entered it.
        // A better way would be to store the raw passphrase in memory IF obtained via prompt for the session.
        // For this iteration, we will re-prompt for save if not in ENV.
        fmt.Fprintln(os.Stderr, "Re-enter passphrase to confirm save operation:")
        currentPassphrase, errPass = getPassphrase(false) // false = don't need double confirm, just get it.
        if errPass != nil || currentPassphrase == "" {
            return fmt.Errorf("passphrase re-confirmation failed for saving: %w", errPass)
        }
    }


	plaintext, err := json.MarshalIndent(satellitesData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal satellite data for encryption: %w", err)
	}

	// Always generate a new salt for each save for maximum security.
	salt := make([]byte, config.Argon2SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key with the current passphrase and the NEW salt
	keyForSave, keyErr := crypto.DeriveKeyWithArgon2id(currentPassphrase, salt)
	if keyErr != nil {
		return fmt.Errorf("key derivation for save failed: %w", keyErr)
	}
	// Update the session key. This is the key corresponding to the current file state.
	sessionKey = keyForSave

	// Encrypt using the derived key (nonce will be generated by crypto.Encrypt)
	nonceAndCiphertext, err := crypto.Encrypt(plaintext, keyForSave)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	// Prepend salt to the (nonce + ciphertext) payload
	encryptedFileBytes := append(salt, nonceAndCiphertext...)

	// Write to a temporary file first for atomicity
	tempDataPath := dataPath + ".tmp"
	if err := ioutil.WriteFile(tempDataPath, encryptedFileBytes, 0600); err != nil { // 0600 for restricted permissions
		return fmt.Errorf("failed to write temporary encrypted datastore %s: %w", tempDataPath, err)
	}

	// Atomically replace the old file with the new one
	if err := os.Rename(tempDataPath, dataPath); err != nil {
		// Attempt to clean up temp file if rename fails
		_ = os.Remove(tempDataPath)
		return fmt.Errorf("failed to commit encrypted datastore from %s to %s: %w", tempDataPath, dataPath, err)
	}
	return nil
}


