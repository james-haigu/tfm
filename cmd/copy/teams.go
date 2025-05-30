// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package copy

import (
	"fmt"

	"strings"

	"github.com/hashicorp-services/tfm/output"
	"github.com/hashicorp-services/tfm/tfclient"
	tfe "github.com/hashicorp/go-tfe"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	o output.Output

	// `tfemig copy teams` command
	teamCopyCmd = &cobra.Command{
		Use:   "teams",
		Short: "Copy Teams",
		Long:  "Copy Teams from source to destination org",
		RunE: func(cmd *cobra.Command, args []string) error {
			return copyTeams(
				tfclient.GetClientContexts())

		},
		PostRun: func(cmd *cobra.Command, args []string) {
			o.Close()
		},
	}
)

func init() {

	// Add commands
	CopyCmd.AddCommand(teamCopyCmd)

}

// Get all source target Teams
func discoverSrcTeams(c tfclient.ClientContexts) ([]*tfe.Team, error) {
	o.AddMessageUserProvided("Getting list of teams from source TFE: ", c.SourceHostname)
	srcTeams := []*tfe.Team{}

	opts := tfe.TeamListOptions{
		ListOptions: tfe.ListOptions{
			PageNumber: 1,
			PageSize:   100},
	}

	for {
		items, err := c.SourceClient.Teams.List(c.SourceContext, c.SourceOrganizationName, &opts)
		if err != nil {
			return nil, err
		}

		srcTeams = append(srcTeams, items.Items...)

		o.AddFormattedMessageCalculated("Found %d Teams in source", len(srcTeams))

		if items.CurrentPage >= items.TotalPages {
			break
		}
		opts.PageNumber = items.NextPage

	}

	return srcTeams, nil
}

func getSrcTeamsFilter(c tfclient.ClientContexts, tmList []string) ([]*tfe.Team, error) {
	srcTeams := []*tfe.Team{}
	found_flag := false
	pn := 1

	fmt.Println("Received team name list:", tmList)

	for _, tm := range tmList {
		fmt.Println("Processing team:", tm)
		pn = 1
		found_flag = false
		for {

			opts := tfe.TeamListOptions{
				ListOptions: tfe.ListOptions{
					PageNumber: pn,
					PageSize:   100},
				Query: tm, // Seems this option doesn't work yet. List call will always return full team list
			}

			// This always return the full team list
			items, err := c.SourceClient.Teams.List(c.SourceContext, c.SourceOrganizationName, &opts)
			if err != nil {
				return nil, err
			}

			item_len := len(items.Items)
			fmt.Printf("Processing %d of %d batches (next batch %d)\n", items.CurrentPage, items.TotalPages, items.NextPage)
			fmt.Printf("Retrieved %d teams from source TFE org %s\n", item_len, c.SourceOrganizationName)

			// If multiple teams returned, find exact match
			indexMatch := 0
			if item_len >= 1 {
				indexMatch = 0
				for _, result := range items.Items {
					//fmt.Printf("Try to match team name (%s) with %s\n", tm, result.Name)
					if tm == result.Name {
						// Found matching team name
						srcTeams = append(srcTeams, items.Items[indexMatch])
						fmt.Println("Found matching team name: ", tm, "in source")
						found_flag = true
						break
					}
					indexMatch++
				}
			} else {
				o.AddMessageUserProvided2("Warning:", "Did NOT find matching team in source", tm)
			}

			if indexMatch >= item_len {
				o.AddMessageUserProvided2("Warning:", fmt.Sprintf("Did NOT find matching team in batch %d", items.CurrentPage), tm)
			}

			if items.CurrentPage >= items.TotalPages || found_flag {
				break
			}

			pn = items.NextPage

		}
	}

	return srcTeams, nil
}

// Gets all teams defined in the configuration file `teams` lists from the source TFE
func getSrcTeamsCfg(c tfclient.ClientContexts) ([]*tfe.Team, error) {
	var srcTeams []*tfe.Team
	var err error

	// Get source teams list from config list `teams` if it exists
	srcTeamsCfg := viper.GetStringSlice("teams")

	if len(srcTeamsCfg) > 0 {
		o.AddFormattedMessageCalculated("Found %d teams in `teams` config", len(srcTeamsCfg))
		fmt.Println("Using teams listed in config:", srcTeamsCfg)

		//get source teams
		srcTeams, err = getSrcTeamsFilter(tfclient.GetClientContexts(), srcTeamsCfg)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to list Teams from config")
		}
		// If no teams found in config (list), get all teams from source
	} else {
		// Get ALL source teams
		fmt.Println("ALL TEAMS WILL BE MIGRATED from ", viper.GetString("src_tfe_hostname"))

		srcTeams, err = discoverSrcTeams(tfclient.GetClientContexts())
		if err != nil {
			return nil, errors.Wrap(err, "Failed to list Teams from source")
		}
	}

	fmt.Printf("%d teams will be migrated\n", len(srcTeams))
	return srcTeams, nil
}

// Get all destination target Teams
func discoverDestTeams(c tfclient.ClientContexts) ([]*tfe.Team, error) {
	o.AddMessageUserProvided("Getting list of teams from destination TFE: ", c.DestinationHostname)
	destTeams := []*tfe.Team{}

	opts := tfe.TeamListOptions{
		ListOptions: tfe.ListOptions{
			PageNumber: 1,
			PageSize:   100},
	}
	for {
		items, err := c.DestinationClient.Teams.List(c.DestinationContext, c.DestinationOrganizationName, &opts)
		if err != nil {
			return nil, err
		}

		destTeams = append(destTeams, items.Items...)

		o.AddFormattedMessageCalculated("Found %d Teams in destination", len(destTeams))

		if items.CurrentPage >= items.TotalPages {
			break
		}
		opts.PageNumber = items.NextPage

	}

	return destTeams, nil
}

// Takes a team name and a slice of teams as type []*tfe.Team and
// returns true if the team name exists within the provided slice of teams.
// Used to compare source team names to the existing destination team names.
func doesTeamExist(teamName string, teams []*tfe.Team) bool {
	// Convert the teamName to lowercase (or uppercase if you prefer) for case-insensitive comparison
	teamName = strings.ToLower(teamName)

	for _, t := range teams {
		// Convert the team name in the teams slice to lowercase (or uppercase) for comparison
		if teamName == strings.ToLower(t.Name) {
			return true
		}
	}
	return false
}

// Gets all source team names and all destination team names and recreates
// the source teams in the destination if the team name does not exist in the destination.
func copyTeams(c tfclient.ClientContexts) error {
	// Get the source teams properties
	srcTeams, err := getSrcTeamsCfg(tfclient.GetClientContexts())
	if err != nil {
		return errors.Wrap(err, "failed to list teams from source")
	}

	// Get the destination teams properties
	destTeams, err := discoverDestTeams(tfclient.GetClientContexts())
	if err != nil {
		return errors.Wrap(err, "failed to list teams from destination")
	}

	// Loop each team in the srcTeams slice, check for the team existence in the destination,
	// and if a team exists in the destination, then do nothing, else create team in destination.
	for _, srcteam := range srcTeams {
		exists := doesTeamExist(srcteam.Name, destTeams)
		if exists {
			o.AddMessageUserProvided2("Warning:", "SKIP migration - Existed in destination", srcteam.Name)
		} else {
			fmt.Println("Migrating ", srcteam.Name)
			retSrcteam, err := c.DestinationClient.Teams.Create(c.DestinationContext, c.DestinationOrganizationName, tfe.TeamCreateOptions{
				Type:      "",
				Name:      &srcteam.Name,
				SSOTeamID: &srcteam.SSOTeamID,
				OrganizationAccess: &tfe.OrganizationAccessOptions{
					ManagePolicies:        &srcteam.OrganizationAccess.ManagePolicies,
					ManagePolicyOverrides: &srcteam.OrganizationAccess.ManagePolicyOverrides,
					ManageWorkspaces:      &srcteam.OrganizationAccess.ManageWorkspaces,
					ManageVCSSettings:     &srcteam.OrganizationAccess.ManageVCSSettings,
					ManageProviders:       &srcteam.OrganizationAccess.ManageProviders,
					ManageModules:         &srcteam.OrganizationAccess.ManageModules,
					ManageRunTasks:        &srcteam.OrganizationAccess.ManageRunTasks,
					// release v202302-1
					ManageProjects: &srcteam.OrganizationAccess.ManageProjects,
					ReadWorkspaces: &srcteam.OrganizationAccess.ReadWorkspaces,
					ReadProjects:   &srcteam.OrganizationAccess.ReadProjects,
					// release 202303-1
					ManageMembership: &srcteam.OrganizationAccess.ManageMembership,
				},
				Visibility: &srcteam.Visibility,
			})
			if err != nil {
				return err
			}
			if retSrcteam != nil {
				fmt.Printf("ok\n")
			}
			o.AddDeferredMessageRead("Migrated ", srcteam.Name)
		}
	}
	return nil
}
