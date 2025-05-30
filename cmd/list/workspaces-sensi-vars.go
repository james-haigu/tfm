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

	// `tfm list workspaces-sensi-vars` command
	workspacesSensiVarsListCmd = &cobra.Command{
		Use:     "workspaces-sensi-vars",
		Aliases: []string{"ws-dtls"},
		Short:   "Workspaces sensitive vars command",
		Long:    "List Workspaces sensitive vars in an org",
		Run: func(cmd *cobra.Command, args []string) {
			listWorkspacesSensiVars(tfclient.GetClientContexts())
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			o.Close()
		},
	}
)

func init() {

	// Add commands
	ListCmd.AddCommand(workspacesSensiVarsListCmd)

}

func listWorkspacesSensiVars(c tfclient.ClientContexts) error {

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
	o.AddTableHeaders("Seq", "Workspace", "VarSeq", "Var", "Sensitive")

	for i, ws := range workspacesList {
		var_list, err := getWorkspaceSensiVars(client, cntxt, ws.ID)
		if err != nil {
			return err
		}

		if len(var_list.Items) == 0 {
			o.AddTableRows(i, ws.Name, 0, "na", "")
		} else {
			for seq, v := range var_list.Items {
				s := ""
				if v.Sensitive {
					s = "(s)"
				}
				o.AddTableRows(i+1, ws.Name, seq+1, v.Key, s)
			}
		}
	}

	return nil
}

// return a string contains list of varaibles and mark sensible vars
func getWorkspaceSensiVars(client *tfe.Client, ctx context.Context, wsID string) (*tfe.VariableList, error) {
	// will NOT implement loop throught pages because
	// not expecing more than 100 variables in any workspace
	opts := tfe.VariableListOptions{
		ListOptions: tfe.ListOptions{
			PageNumber: 1,
			PageSize:   100},
	}

	var_list, err := client.Variables.List(ctx, wsID, &opts)
	if err != nil {
		return nil, err
	}

	return var_list, err
}
