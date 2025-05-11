// cmd/satcli/table_printer.go
package main

// Currently, printSatellitesTable is in main.go.
// It could be moved here for better organization if main.go grows too large.
// Remember to adjust visibility (e.g., make it public if called from other packages,
// or keep it private if only used within the main cmd package).
/*
import (
	"fmt"
	"os"
	"text/tabwriter"
	"github.com/yackko/satcom-code/types" // Adjust import path
)

func PrintSatellitesTable(satellitesToPrint []types.Satellite) {
	if len(satellitesToPrint) == 0 { return }
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tOPERATOR\tSTATUS\tORBIT TYPE\tLAUNCH DATE\tALTITUDE (km)\tCONSTELLATION")
	fmt.Fprintln(w, "----\t--------\t------\t----------\t-----------\t-------------\t-------------")
	for _, sat := range satellitesToPrint {
		constellationStatus := "No"; if sat.Constellation { constellationStatus = "Yes" }
		altitudeStr := fmt.Sprintf("%.0f", sat.Altitude)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			sat.Name, sat.Operator, sat.Status, sat.OrbitType, sat.LaunchDate, altitudeStr, constellationStatus)
	}
	w.Flush()
}
*/
