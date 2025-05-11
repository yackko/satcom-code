// cmd/satcli/main.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	// Adjust module path if different from "satcom-code"
	"github.com/yackko/satcom-code/internal/config"
	"github.com/yackko/satcom-code/internal/datastore"
	"github.com/yackko/satcom-code/tui" // For the TUI list view
	"github.com/yackko/satcom-code/types"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// orbitExplanations map - specific to explainCmd in this package
var orbitExplanations = map[string]string{
	"LEO": "Low Earth Orbit (LEO):\n  Altitude: Typically 160 to 2,000 kilometers (100 to 1,240 miles).\n  Characteristics: Short orbital periods (around 90 minutes to 2 hours). Satellites move quickly relative to the Earth's surface.\n  Uses: Earth observation, remote sensing, communications (e.g., Starlink), International Space Station (ISS).\n  Pros: Lower launch costs, lower signal latency.\n  Cons: Limited coverage from a single satellite (requires constellations for continuous coverage), atmospheric drag can be a factor at lower LEO altitudes.",
	"MEO": "Medium Earth Orbit (MEO):\n  Altitude: Between LEO and GEO, typically from 2,000 km up to 35,786 km (just below geostationary).\n  Common Altitudes: Around 20,200 km for navigation satellites.\n  Characteristics: Orbital periods of a few hours (e.g., 12 hours for GPS).\n  Uses: Navigation systems (e.g., GPS, GLONASS, Galileo), some communications.\n  Pros: Wider coverage than LEO, lower latency than GEO.\n  Cons: Fewer satellites needed than LEO for global coverage, but more than GEO.",
	"GEO": "Geostationary Orbit (GEO) / Geosynchronous Equatorial Orbit:\n  Altitude: Precisely 35,786 kilometers (22,236 miles) directly above the Earth's Equator.\n  Characteristics: Orbital period matches Earth's rotation (23 hours, 56 minutes, 4 seconds). Satellites appear stationary from the ground.\n  Uses: Telecommunications (broadcast TV, fixed communications), weather monitoring (e.g., GOES).\n  Pros: Wide coverage area (one satellite can cover about 1/3 of Earth's surface), fixed ground antennas.\n  Cons: Significant signal latency due to high altitude, higher launch costs, poor coverage for polar regions.",
	"GSO": "Geosynchronous Orbit (GSO):\n  Altitude: Also 35,786 kilometers.\n  Characteristics: Orbital period matches Earth's rotation. However, unlike GEO, GSO orbits can be inclined. A satellite in GSO will return to the same position in the sky at the same time each day, but it will appear to trace a path (an analemma) if inclined.\n  Uses: Similar to GEO; some communications and broadcasting.\n  Note: GEO is a special case of GSO where the inclination is zero.",
	"HEO": "Highly Elliptical Orbit (HEO):\n  Characteristics: Orbit with a low perigee (closest point to Earth) and a very high apogee (farthest point). Satellites spend most of their time near apogee, moving slowly over a specific region.\n  Uses: Communications and broadcasting for high-latitude regions (e.g., Molniya orbits for Russia, SiriusXM radio satellites using Tundra orbits), some scientific missions.\n  Pros: Long dwell time over specific areas, good for covering regions not well served by GEO.\n  Cons: Requires steerable ground antennas, varying distance to satellite.",
	"SSO": "Sun-Synchronous Orbit (SSO):\n  Characteristics: A type of polar orbit where the satellite passes over any given point on Earth's surface at the same local solar time. This means lighting conditions are consistent for imaging.\n  Altitude: Typically LEO altitudes (e.g., 600-800 km).\n  Inclination: Near-polar (e.g., around 98 degrees).\n  Uses: Earth observation, environmental monitoring, reconnaissance, weather satellites.\n  Pros: Consistent illumination for imaging and change detection.\n  Cons: Similar to LEO in terms of coverage per satellite.",
	"HALO": "Halo Orbit:\n  Characteristics: A periodic, three-dimensional orbit near one of the Lagrange points (L1, L2, or L3) in a two-body system (e.g., Earth-Sun or Earth-Moon). These orbits don't orbit a celestial body directly but rather a point in space where gravitational forces balance.\n  Uses: Space telescopes (e.g., James Webb Space Telescope at Sun-Earth L2, SOHO at Sun-Earth L1), scientific observation, potential communication relays.\n  Pros: Provides a stable vantage point for observing the Earth, Sun, or deep space with minimal obstruction or interference. Can offer continuous view of certain regions.\n  Cons: Inherently unstable for some Lagrange points, requiring station-keeping maneuvers.",
}

// printSatellitesTable formats and prints a list of satellites as a table.
// This function remains in main.go (or could be moved to table_printer.go).
func printSatellitesTable(satellitesToPrint []types.Satellite) {
	if len(satellitesToPrint) == 0 {
		return // Caller should ideally handle "no results found" message
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tOPERATOR\tSTATUS\tORBIT TYPE\tLAUNCH DATE\tALTITUDE (km)\tCONSTELLATION")
	fmt.Fprintln(w, "----\t--------\t------\t----------\t-----------\t-------------\t-------------")
	for _, sat := range satellitesToPrint {
		constellationStatus := "No"
		if sat.Constellation {
			constellationStatus = "Yes"
		}
		altitudeStr := fmt.Sprintf("%.0f", sat.Altitude)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			sat.Name, sat.Operator, sat.Status, sat.OrbitType, sat.LaunchDate, altitudeStr, constellationStatus)
	}
	w.Flush()
}

var rootCmd = &cobra.Command{
	Use:   "satcli",
	Short: "Satcli is a CLI tool for managing and querying satellite information.",
	Long: `Satcli provides a command-line interface to manage a local, secure datastore of Earth satellites.
If the ` + config.PassphraseEnvVar + ` environment variable is not set, you will be prompted for a passphrase.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "help" || cmd.CalledAs() == "help" || // Check for 'help' subcommand itself
           (cmd.Parent() != nil && cmd.Parent().Name() == "help") || // Check if parent is 'help' (for subcommands of help)
			cmd.Name() == "version" || cmd.CalledAs() == "version" ||
			strings.HasPrefix(cmd.Use, "completion") { // Check Use field for completion
			return nil
		}
		if err := datastore.Init(); err != nil {
			if !strings.Contains(err.Error(), "passphrase") && !strings.Contains(err.Error(), "decrypt") && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Critical error during datastore initialization: %v\n", err)
				return err
			}
			// Non-critical init errors (like passphrase prompt failed for non-existent file) are handled by datastore.Init printing a notice.
			// Individual commands will check datastore.IsUnlocked().
		}
		return nil
	},
}

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query satellites based on specified criteria from the secure datastore",
	Long: `Query satellites from the local, secure datastore using a combination of criteria.
If ` + config.PassphraseEnvVar + ` is not set, you will be prompted for the passphrase.
Supports filtering by operator, status, orbit type, launch dates, constellation status, and altitude.
Output can be formatted as JSON (default), table, or an interactive TUI.

Examples:
  satcli query --operator ESA --status active --orbit-type LEO --output tui
  satcli query --launch-after 2022-01-01 --constellation true --output table`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !datastore.IsUnlocked() {
			return fmt.Errorf("datastore not accessible. Passphrase not provided or was incorrect. Set %s or enter correct passphrase at prompt.", config.PassphraseEnvVar)
		}
		satsMap, err := datastore.GetSatellites()
		if err != nil {
			return fmt.Errorf("failed to get satellites: %w", err)
		}

		operatorFilter, _ := cmd.Flags().GetString("operator")
		statusFilter, _ := cmd.Flags().GetString("status")
		orbitTypeFilter, _ := cmd.Flags().GetString("orbit-type")
		launchAfterStr, _ := cmd.Flags().GetString("launch-after")
		launchBeforeStr, _ := cmd.Flags().GetString("launch-before")
		constellationStr, _ := cmd.Flags().GetString("constellation")
		minAltitude, _ := cmd.Flags().GetFloat64("min-altitude")
		maxAltitude, _ := cmd.Flags().GetFloat64("max-altitude")
		outputFormat, _ := cmd.Flags().GetString("output")

		var launchAfterDate, launchBeforeDate time.Time
		if launchAfterStr != "" {
			var errParse error
			launchAfterDate, errParse = time.Parse(config.DateFormat, launchAfterStr)
			if errParse != nil { cmd.SilenceUsage = true; return fmt.Errorf("invalid format for --launch-after: '%s'. Use YYYY-MM-DD. (Details: %w)", launchAfterStr, errParse) }
		}
		if launchBeforeStr != "" {
			var errParse error
			launchBeforeDate, errParse = time.Parse(config.DateFormat, launchBeforeStr)
			if errParse != nil { cmd.SilenceUsage = true; return fmt.Errorf("invalid format for --launch-before: '%s'. Use YYYY-MM-DD. (Details: %w)", launchBeforeStr, errParse) }
		}
		if !launchAfterDate.IsZero() && !launchBeforeDate.IsZero() && launchAfterDate.After(launchBeforeDate) {
			cmd.SilenceUsage = true; return fmt.Errorf("--launch-after date (%s) cannot be after --launch-before date (%s)", launchAfterStr, launchBeforeStr)
		}
		if minAltitude > 0 && maxAltitude > 0 && minAltitude > maxAltitude {
			cmd.SilenceUsage = true; return fmt.Errorf("--min-altitude (%.0f) cannot be greater than --max-altitude (%.0f)", minAltitude, maxAltitude)
		}

		var filteredSatellites []types.Satellite
		for _, sat := range satsMap {
			matches := true
			if operatorFilter != "" && !strings.EqualFold(sat.Operator, operatorFilter) { matches = false }
			if matches && statusFilter != "" && !strings.EqualFold(sat.Status, statusFilter) { matches = false }
			if matches && orbitTypeFilter != "" && !strings.EqualFold(sat.OrbitType, orbitTypeFilter) { matches = false }
			if matches && (launchAfterStr != "" || launchBeforeStr != "") {
				satLaunchDate, errDateParse := time.Parse(config.DateFormat, sat.LaunchDate)
				if errDateParse != nil { matches = false
				} else {
					if !launchAfterDate.IsZero() && satLaunchDate.Before(launchAfterDate) { matches = false }
					if matches && !launchBeforeDate.IsZero() && satLaunchDate.After(launchBeforeDate) { matches = false }
				}
			}
			if !matches { continue }
			if constellationStr != "" {
				constellationFilterVal, errBoolParse := strconv.ParseBool(constellationStr)
				if errBoolParse != nil { cmd.SilenceUsage = true; return fmt.Errorf("invalid value for --constellation: '%s'. Use 'true' or 'false'", constellationStr) }
				if sat.Constellation != constellationFilterVal { matches = false }
			}
			if !matches { continue }
			if minAltitude > 0 && sat.Altitude < minAltitude { matches = false }
			if matches && maxAltitude > 0 && sat.Altitude > maxAltitude { matches = false }
			if matches { filteredSatellites = append(filteredSatellites, sat) }
		}
		sort.Slice(filteredSatellites, func(i, j int) bool { return filteredSatellites[i].Name < filteredSatellites[j].Name })

		if len(filteredSatellites) == 0 {
			fmt.Println("No satellites found matching specified criteria.")
			return nil
		}
		
		fmt.Printf("Found %d matching satellite(s).\n", len(filteredSatellites))
		switch strings.ToLower(outputFormat) {
		case "tui":
			model := tui.NewListModel(filteredSatellites) // From tui package
			p := tea.NewProgram(model, tea.WithAltScreen())
			if _, errRun := p.Run(); errRun != nil {
				return fmt.Errorf("error running TUI: %w", errRun)
			}
		case "table":
			printSatellitesTable(filteredSatellites)
		default: // JSON
			output, errJson := json.MarshalIndent(filteredSatellites, "", "  ")
			if errJson != nil { return fmt.Errorf("failed to marshal filtered satellites to JSON: %w", errJson) }
			fmt.Println(string(output))
		}
		return nil
	},
}

var addCmd = &cobra.Command{
	Use:   "add [name] [operator] [status] [orbitType]",
	Short: "Add a new satellite record to the secure datastore",
	Long:  "Adds a new satellite with essential information. If " + config.PassphraseEnvVar + " is not set, you will be prompted.",
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !datastore.IsUnlocked() {
			return fmt.Errorf("datastore not accessible. Passphrase not provided or was incorrect. Set %s or enter correct passphrase at prompt.", config.PassphraseEnvVar)
		}
		name, operator, status, orbitType := args[0], args[1], args[2], args[3]
		if name == "" { cmd.SilenceUsage = true; return fmt.Errorf("satellite name cannot be empty") }

		newSat := types.Satellite{
			Name: name, Operator: operator, Status: status, OrbitType: orbitType,
			// Consider prompting for more fields or using flags for a richer 'add' experience
			LaunchDate: time.Now().Format(config.DateFormat), // Default launch date to today
		}
		if err := datastore.AddSatellite(newSat); err != nil { // Pass the whole struct
			cmd.SilenceUsage = true 
			return err // AddSatellite will give specific error (e.g., duplicate)
		}
		if err := datastore.Save(); err != nil {
			return fmt.Errorf("failed to save record for '%s': %w", name, err)
		}
		fmt.Printf("Record added: %s (encrypted in datastore)\n", name)
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all satellite records from the secure datastore",
	Long:  "Retrieves and displays all satellite records. If " + config.PassphraseEnvVar + " is not set, you will be prompted.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !datastore.IsUnlocked() {
			return fmt.Errorf("datastore not accessible. Passphrase not provided or was incorrect. Set %s or enter correct passphrase at prompt.", config.PassphraseEnvVar)
		}
		satsMap, err := datastore.GetSatellites()
		if err != nil {
			return fmt.Errorf("failed to get satellites: %w", err)
		}
        if len(satsMap) == 0 {
             fmt.Println("Datastore is accessible but contains no satellite records.")
             return nil
        }
		outputFormat, _ := cmd.Flags().GetString("output")
		var satList []types.Satellite
		for _, sat := range satsMap { satList = append(satList, sat) }
		sort.Slice(satList, func(i, j int) bool { return satList[i].Name < satList[j].Name })
		
		fmt.Printf("Total records: %d.\n", len(satList))
		switch strings.ToLower(outputFormat) {
		case "tui":
			model := tui.NewListModel(satList) // From tui package
			p := tea.NewProgram(model, tea.WithAltScreen())
			if _, errRun := p.Run(); errRun != nil {
				return fmt.Errorf("error running TUI: %w", errRun)
			}
		case "table":
			printSatellitesTable(satList)
		default: // JSON
			output, errJson := json.MarshalIndent(satList, "", "  ")
			if errJson != nil { return fmt.Errorf("failed to marshal satellite list to JSON: %w", errJson) }
			fmt.Println(string(output))
		}
		return nil
	},
}

var explainCmd = &cobra.Command{
	Use:   "explain [category] [term]",
	Short: "Explain a specific term or concept related to satellites.",
	Long: `Provides a definition or explanation for various terms. Currently supports explaining 'orbit' types.
Examples:
  satcli explain orbit LEO
  satcli explain orbit GEO`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		category := strings.ToLower(args[0])
		term := strings.ToUpper(args[1])
		if category == "orbit" {
			if explanation, found := orbitExplanations[term]; found {
				fmt.Println(explanation)
			} else {
				fmt.Fprintf(os.Stderr, "Error: Unknown orbit type: %s\n", term)
				fmt.Fprintln(os.Stderr, "Supported orbit types are:")
				var supportedOrbits []string
				for k := range orbitExplanations { supportedOrbits = append(supportedOrbits, k) }
				sort.Strings(supportedOrbits)
				for _, t := range supportedOrbits { fmt.Fprintf(os.Stderr, "  - %s\n", t) }
				cmd.SilenceUsage = true; return fmt.Errorf("explanation not found for orbit type '%s'", args[1])
			}
		} else {
			cmd.SilenceUsage = true; return fmt.Errorf("unknown category for explanation: '%s'. Currently, only 'orbit' category is supported", category)
		}
		return nil
	},
}

func init() {
	queryCmd.Flags().StringP("operator", "o", "", "Filter by satellite operator (case-insensitive)")
	queryCmd.Flags().StringP("status", "s", "", "Filter by satellite status (case-insensitive)")
	queryCmd.Flags().StringP("orbit-type", "t", "", "Filter by orbit type (e.g., LEO, GEO; case-insensitive)")
	queryCmd.Flags().String("launch-after", "", "Filter satellites launched after this date (YYYY-MM-DD)")
	queryCmd.Flags().String("launch-before", "", "Filter satellites launched before this date (YYYY-MM-DD)")
	queryCmd.Flags().String("constellation", "", "Filter by constellation status ('true' or 'false')")
	queryCmd.Flags().Float64("min-altitude", 0, "Filter by minimum altitude in km (0 means no filter)")
	queryCmd.Flags().Float64("max-altitude", 0, "Filter by maximum altitude in km (0 means no filter)")
	queryCmd.Flags().StringP("output", "O", "json", "Output format: json, table, or tui")

	listCmd.Flags().StringP("output", "O", "json", "Output format: json, table, or tui")
    addCmd.Flags().Bool("encrypt-check", true, "dummy flag to ensure addCmd has one for example")


	rootCmd.AddCommand(queryCmd, addCmd, listCmd, explainCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
