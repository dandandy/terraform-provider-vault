package vault

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/vault/api"
)

func TestLDAPAuthBackend_basic(t *testing.T) {
	path := acctest.RandomWithPrefix("tf-test-ldap-path")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testProviders,
		CheckDestroy: testLDAPAuthBackendDestroy,
		Steps: []resource.TestStep{
			{
				Config: testLDAPAuthBackendConfig_basic(path),
				Check:  testLDAPAuthBackendCheck_attrs(path),
			},
		},
	})
}

func testLDAPAuthBackendDestroy(s *terraform.State) error {
	client := testProvider.Meta().(*api.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vault_ldap_auth_backend" {
			continue
		}
		secret, err := client.Logical().Read(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error checking for LDAP auth backend %q: %s", rs.Primary.ID, err)
		}
		if secret != nil {
			return fmt.Errorf("LDAP auth backend %q still exists", rs.Primary.ID)
		}
	}
	return nil
}

func testLDAPAuthBackendCheck_attrs(path string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState := s.Modules[0].Resources["vault_ldap_auth_backend.test"]
		if resourceState == nil {
			return fmt.Errorf("resource not found in state")
		}

		instanceState := resourceState.Primary
		if instanceState == nil {
			return fmt.Errorf("resource has no primary instance")
		}

		endpoint := strings.Trim(path, "/")
		if endpoint != instanceState.ID {
			return fmt.Errorf("expected ID to be %q, got %q instead", endpoint, instanceState.ID)
		}

		client := testProvider.Meta().(*api.Client)
		authMounts, err := client.Sys().ListAuth()
		if err != nil {
			return err
		}
		authMount := authMounts[strings.Trim(path, "/")+"/"]

		if authMount == nil {
			return fmt.Errorf("auth mount %s not present", path)
		}

		if "ldap" != authMount.Type {
			return fmt.Errorf("incorrect mount type: %s", authMount.Type)
		}

		configPath := "auth/" + endpoint + "/config"

		resp, err := client.Logical().Read(configPath)
		if err != nil {
			return err
		}

		attrs := map[string]string{
			"url":             "url",
			"starttls":        "starttls",
			"tls_min_version": "tls_min_version",
			"tls_max_version": "tls_max_version",
			"insecure_tls":    "insecure_tls",
			"certificate":     "certificate",
			"binddn":          "binddn",
			"bindpass":        "bindpass",
			"userdn":          "userdn",
			"userattr":        "userattr",
			"discoverdn":      "discoverdn",
			"deny_null_bind":  "deny_null_bind",
			"upndomain":       "upndomain",
			"groupfilter":     "groupfilter",
			"groupdn":         "groupdn",
			"groupattr":       "groupattr",
		}

		for stateAttr, apiAttr := range attrs {
			if resp.Data[apiAttr] == nil && instanceState.Attributes[stateAttr] == "" {
				continue
			}
			var match bool
			switch resp.Data[apiAttr].(type) {
			case json.Number:
				apiData, err := resp.Data[apiAttr].(json.Number).Int64()
				if err != nil {
					return fmt.Errorf("Expected API field %s to be an int, was %q", apiAttr, resp.Data[apiAttr])
				}
				stateData, err := strconv.ParseInt(instanceState.Attributes[stateAttr], 10, 64)
				if err != nil {
					return fmt.Errorf("Expected state field %s to be an int, was %q", stateAttr, instanceState.Attributes[stateAttr])
				}
				match = apiData == stateData
			case bool:
				if _, ok := resp.Data[apiAttr]; !ok && instanceState.Attributes[stateAttr] == "" {
					match = true
				} else {
					stateData, err := strconv.ParseBool(instanceState.Attributes[stateAttr])
					if err != nil {
						return fmt.Errorf("Expected state field %s to be a bool, was %q", stateAttr, instanceState.Attributes[stateAttr])
					}
					match = resp.Data[apiAttr] == stateData
				}

			case []interface{}:
				apiData := resp.Data[apiAttr].([]interface{})
				length := instanceState.Attributes[stateAttr+".#"]
				if length == "" {
					if len(resp.Data[apiAttr].([]interface{})) != 0 {
						return fmt.Errorf("Expected state field %s to have %d entries, had 0", stateAttr, len(apiData))
					}
					match = true
				} else {
					count, err := strconv.Atoi(length)
					if err != nil {
						return fmt.Errorf("Expected %s.# to be a number, got %q", stateAttr, instanceState.Attributes[stateAttr+".#"])
					}
					if count != len(apiData) {
						return fmt.Errorf("Expected %s to have %d entries in state, has %d", stateAttr, len(apiData), count)
					}
					for i := 0; i < count; i++ {
						stateData := instanceState.Attributes[stateAttr+"."+strconv.Itoa(i)]
						if stateData != apiData[i] {
							return fmt.Errorf("Expected item %d of %s (%s in state) of %q to be %q, got %q", i, apiAttr, stateAttr, endpoint, stateData, apiData[i])
						}
					}
					match = true
				}
			default:
				match = resp.Data[apiAttr] == instanceState.Attributes[stateAttr]

			}
			if !match {
				return fmt.Errorf("Expected %s (%s in state) of %q to be %q, got %q", apiAttr, stateAttr, endpoint, instanceState.Attributes[stateAttr], resp.Data[apiAttr])
			}

		}

		return nil
	}
}

func testLDAPAuthBackendConfig_basic(path string) string {

	return fmt.Sprintf(`
resource "vault_ldap_auth_backend" "test" {
    path                   = "%s"
    url                    = "ldaps://example.org"
    starttls               = true
    tls_min_version        = "tls11"
    tls_max_version        = "tls12"
    insecure_tls           = false
}
`, path)

}
