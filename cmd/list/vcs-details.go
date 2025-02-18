// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package list

import (
	"github.com/hashicorp-services/tfm/cmd/helper"
	"github.com/hashicorp-services/tfm/tfclient"
	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/cobra"
)

var (
	//vcsOutput output.Output

	vcsDetailsListCmd = &cobra.Command{
		Use:     "vcs-details",
		Aliases: []string{"vcs-details"},
		Short:   "List VCS Providers Details",
		Long:    "List of VCS Providers Details. Will default to source if no side is specified",
		Run: func(cmd *cobra.Command, args []string) {
			vcsDetailsList(tfclient.GetClientContexts())
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			o.Close()
		},
	}
)

func init() {
	ListCmd.AddCommand(vcsDetailsListCmd)
}

// helper functions
// Get vcs details for a given Organization
func vcsDetailsListAllForOrganization(c tfclient.ClientContexts, orgName string) ([]*tfe.OAuthClient, error) {
	var allItems []*tfe.OAuthClient
	opts := tfe.OAuthClientListOptions{
		ListOptions: tfe.ListOptions{PageNumber: 1, PageSize: 100},
	}

	// Loop through pages in case found more than 100 vcs (not likely!)
	for {
		var items *tfe.OAuthClientList
		var err error

		if (ListCmd.Flags().Lookup("side").Value.String() == "source") || (!ListCmd.Flags().Lookup("side").Changed) {
			items, err = c.SourceClient.OAuthClients.List(c.SourceContext, orgName, &opts)
		}

		if ListCmd.Flags().Lookup("side").Value.String() == "destination" {
			items, err = c.DestinationClient.OAuthClients.List(c.DestinationContext, orgName, &opts)
		}

		if err != nil {
			return nil, err
		}

		allItems = append(allItems, items.Items...)

		if items.CurrentPage >= items.TotalPages {
			break
		}
		opts.PageNumber = items.NextPage
	}

	return allItems, nil
}

// Get vcs Details
func vcsDetailsList(c tfclient.ClientContexts) error {
	o.AddMessageUserProvided("List vcsDetails for configured Organizations", "")

	var orgVcsList []*tfe.OAuthClient
	var err error
	var vcsDetails *tfe.OAuthClient

	if (ListCmd.Flags().Lookup("side").Value.String() == "source") || (!ListCmd.Flags().Lookup("side").Changed) {
		orgVcsList, err = vcsListAllForOrganization(c, c.SourceOrganizationName)
	}

	if ListCmd.Flags().Lookup("side").Value.String() == "destination" {
		orgVcsList, err = vcsListAllForOrganization(c, c.DestinationOrganizationName)
	}

	if err != nil {
		helper.LogError(err, "failed to list vcs for organization")
	}

	o.AddFormattedMessageCalculated("Found %d vcs", len(orgVcsList))

	// display output as table and put headers as first row
	o.AddTableHeaders("Organization", "Name", "Id", "Service Provider", "Service Provider Name", "Created At", "URL")

	// Loop through the VCS list for this org
	for _, i := range orgVcsList {

		vcsName := "No Name"
		if i.Name != nil {
			vcsName = *i.Name
		}

		OAuthTokenID := "No OAuthToken"
		if i.OAuthTokens != nil {
			OAuthTokenID = i.OAuthTokens[0].ID
		}

		// get details for each VCS
		vcsDetails, err = c.DestinationClient.OAuthClients.List(c.DestinationContext, orgName, &opts)
		o.AddTableRows(i.Organization.Name, vcsName, OAuthTokenID, i.ServiceProvider, i.ServiceProviderName, i.CreatedAt, i.HTTPURL)
	}

	return nil
}
