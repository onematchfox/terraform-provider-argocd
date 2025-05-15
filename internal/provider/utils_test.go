package provider

import (
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testCheckTokenExpiresAt(resourceName string, expiresIn int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("token ID is not set")
		}

		_expiresAt, ok := rs.Primary.Attributes["expires_at"]
		if !ok {
			return fmt.Errorf("expires_at is not set")
		}

		_issuedAt, ok := rs.Primary.Attributes["issued_at"]
		if !ok {
			return fmt.Errorf("testCheckTokenExpiresAt: issued_at is not set")
		}

		expiresAt, err := strconv.ParseInt(_expiresAt, 10, 64)
		if err != nil {
			return fmt.Errorf("testCheckTokenExpiresAt: string attribute 'expires_at' stored in state cannot be converted to int64: %s", err)
		}

		issuedAt, err := strconv.ParseInt(_issuedAt, 10, 64)
		if err != nil {
			return fmt.Errorf("testCheckTokenExpiresAt: string attribute 'issued_at' stored in state cannot be converted to int64: %s", err)
		}

		if issuedAt+expiresIn != expiresAt {
			return fmt.Errorf("testCheckTokenExpiresAt: issuedAt + expiresIn != expiresAt : %d + %d != %d", issuedAt, expiresIn, expiresAt)
		}

		return nil
	}
}

func testTokenIssuedAtSet(name string, count int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		key := "issued_at"

		for i := 0; i < count; i++ {
			ms := s.RootModule()
			_name := fmt.Sprintf("%s.%d", name, i)

			rs, ok := ms.Resources[_name]
			if !ok {
				return fmt.Errorf("not found: %s in %s", _name, ms.Path)
			}

			is := rs.Primary
			if is == nil {
				return fmt.Errorf("no primary instance: %s in %s", _name, ms.Path)
			}

			if val, ok := is.Attributes[key]; !ok || val == "" {
				return fmt.Errorf("%s: Attribute '%s' expected to be set", _name, key)
			}
		}

		return nil
	}
}

func testDelay(seconds int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		time.Sleep(time.Duration(seconds) * time.Second)
		return nil
	}
}
