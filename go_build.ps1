go build -v -x `
-ldflags=" `
-X 'github.com/hashicorp-services/tfm/version.Version=0.11.3' `
-X 'github.com/hashicorp-services/tfm/version.Prerelease=alpha' `
-X 'github.com/hashicorp-services/tfm/version.Build=local-jgu' `
-X 'github.com/hashicorp-services/tfm/version.BuiltBy=James Gu' `
-X 'github.com/hashicorp-services/tfm/version.Date=February 6, 2025 8:59:24 PM' `
"