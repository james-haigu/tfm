// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// I leave this as is for now because the best place to get details is from DB!
// I did learn how to read API document (pkg.go.dev/github.com/hashicorp/go-tfe#Workspace)
// I did learn if the type is defined as 'relation' (such as CurrentRun is defined as a jsonapi:"relation" to Workspace Object),
// then only minimum data (i.e. ID) is available, for details I needed to call API (line 108)

// comparing to list workspaces, workspaces-details was to retrieve more detailed data than already provided
// I have optimized the code to be more concise and explored how add new fields (CurrentRun) to the output

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

	// Set context
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
	o.AddTableHeaders("Name", "Description",
		"ExecutionMode", "VCS Repo", "Auto Apply", "Created At",
		"Locked", "TF Version", "Run Status", "Run Created At")

	for i, ws := range workspacesList {
		ws_repo := "none"
		run_status := "none"
		run_createdAt := ""
		var_details := "none"
		var run *tfe.Run
		var err error

		if ws.VCSRepo != nil {
			ws_repo = ws.VCSRepo.DisplayIdentifier
		}

		if ws.CurrentRun != nil {
			run, err = client.Runs.Read(cntxt, ws.CurrentRun.ID)
			if err != nil {
				return err
			}
			run_status = string(run.Status)
			run_createdAt = run.CreatedAt.Format("2006-01-02 15:01:04")
		}

		v, err := getWorkspaceVars(client, cntxt, ws.ID)
		if err != nil {
			return err
		}
		var_details = v

		o.AddTableRows(ws.Name, ws_repo,
			//ws.Description, ws.ExecutionMode, ws.AutoApply, ws.CreatedAt,
			//ws.Locked, ws.TerraformVersion,
			run_status, run_createdAt, var_details)

		if i == 10 {
			break
		}
	}

	return nil
}

// return a string contains list of varaibles and mark sensible vars
func getWorkspaceVars(client *tfe.Client, ctx context.Context, workspaceID string) (string, error) {
	// will NOT implement loop throught pages because
	// not expecing more than 100 variables in any workspace
	opts := tfe.VariableListOptions{
		ListOptions: tfe.ListOptions{
			PageNumber: 1,
			PageSize:   100},
	}

	var_details := ""

	var_list, err := client.Variables.List(ctx, workspaceID, &opts)
	if err != nil {
		return "", err
	}

	for i, v := range var_list.Items {
		s := ""
		if v.Sensitive {
			s = "(s)"
		}

		if i == 0 {
			var_details = v.Key + s
		} else {
			var_details = var_details + "," + v.Key + s
		}
	}

	return var_details, err
}
