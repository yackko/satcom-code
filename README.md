# Hobby project to learn the Go programming language
# project under development



# SatCLI: Secure Satellite Command-Line Interface

**SatCLI** is a professional-grade command-line application, built with Go, designed for the secure management, advanced querying, and interactive viewing of satellite information. It places a strong emphasis on data security, featuring an encrypted local datastore to protect sensitive satellite records at rest.

Developed for users requiring precise and secure handling of satellite datasets, SatCLI offers a robust and intuitive terminal-based experience.

## Key Features:

* **Secure Encrypted Datastore:** Satellite data is protected using AES-GCM encryption. Encryption keys are derived from a user-provided passphrase via Argon2id, a modern and secure key derivation function. Passphrases are handled via the `SATCLI_PASSPHRASE` environment variable or a secure interactive terminal prompt.
* **Comprehensive Data Operations:**
    * `add`: Securely add new satellite records.
    * `list`: Display all satellite records.
    * `query`: Perform complex, multi-filter queries based on parameters such as operator, status, orbit type, launch date, altitude, and constellation membership.
* **Versatile Output Formats:**
    * **JSON:** Ideal for scripting and interoperability with other tools.
    * **Table:** Clear, human-readable tabular format for quick data review.
    * **TUI (Terminal User Interface):** An interactive view for Browse lists of satellites and viewing detailed information within the terminal, built with Bubble Tea.
* **Informational Commands:**
    * `explain`: Provides definitions and explanations for common satellite-related terms (e.g., orbit types like LEO, GEO, HEO).
* **Professional CLI Experience:**
    * Built with the robust Cobra library for a standard command structure.
    * Clear, concise help messages and user feedback.
    * Organized project structure with distinct packages for types, TUI, and internal logic (configuration, crypto, datastore).

SatCLI aims to be a reliable and secure tool for professionals who manage and analyze specialized satellite datasets directly from their command line.
