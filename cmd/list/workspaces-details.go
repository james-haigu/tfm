// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package list

import (
	"context"
	"fmt"

	"github.com/hashicorp-services/tfm/tfclient"
	"github.com/hashicorp/go-tfe"
	"github.com/spf13/cobra"
)

var (

	// `tfm list workspaces` command
	workspacesDetailsListCmd = &cobra.Command{
		Use:     "workspaces-details",
		Aliases: []string{"ws-dtls"},
		Short:   "Workspaces details command",
		Long:    "List Workspaces details in an org",
		Run: func(cmd *cobra.Command, args []string) {
			listWorkspacesDetails(tfclient.GetClientContexts())
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			o.Close()
		},
	}
)

func init() {

	// Add commands
	ListCmd.AddCommand(workspacesDetailsListCmd)

}

func listWorkspacesDetails(c tfclient.ClientContexts) error {

	workspacesList := []*tfe.Workspace{}
	var client *tfe.Client
	var cntxt context.Context
	hostname := ""
	orgName := ""

	opts := tfe.WorkspaceListOptions{
		ListOptions: tfe.ListOptions{
			PageNumber: 1,
			PageSize:   100},
	}

	// Processing Source side
	if (ListCmd.Flags().Lookup("side").Value.String() == "source") || (!ListCmd.Flags().Lookup("side").Changed) {
		client = c.SourceClient
		cntxt = c.SourceContext
		hostname = c.SourceHostname
		orgName = c.SourceOrganizationName
		o.AddMessageUserProvided("Getting list of workspaces from source TFE: ", hostname)
	} else if ListCmd.Flags().Lookup("side").Value.String() == "destination" {
		client = c.DestinationClient
		cntxt = c.DestinationContext
		hostname = c.DestinationHostname
		orgName = c.DestinationOrganizationName
		o.AddMessageUserProvided("Getting list of workspaces from destination TFE: ", hostname)
	}

	// loop through all pages of workspaces and add them to the workspacesList
	for {
		items, err := client.Workspaces.List(cntxt, orgName, &opts)
		if err != nil {
			fmt.Println("Error With retrieving Workspaces from TFE ", hostname, " : Error ", err)
			return err
		}

		workspacesList = append(workspacesList, items.Items...)

		o.AddFormattedMessageCalculated("Found %d Workspaces", len(workspacesList))

		if items.CurrentPage >= items.TotalPages {
			break
		}
		opts.PageNumber = items.NextPage
	}

	// Output the list of workspaces with details
	o.AddTableHeaders("Name", "Description", "ExecutionMode", "VCS Repo", "Auto Apply", "Created At",
		"Locked", "TF Version", "Current Run")

	for _, i := range workspacesList {
		ws_repo := "none"

		if i.VCSRepo != nil {
			ws_repo = i.VCSRepo.DisplayIdentifier
		}

		o.AddTableRows(i.Name, i.Description, i.ExecutionMode, ws_repo, i.AutoApply, i.CreatedAt,
			i.Locked, i.TerraformVersion, i.CurrentRun.Status)
	}

	return nil
}
