package iam_test

import (
	"fmt"
	"regexp"
	"sort"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	sdkacctest "github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	tfiam "github.com/hashicorp/terraform-provider-aws/internal/service/iam"
)

func TestResourceSortByExpirationDate(t *testing.T) {
	t.Parallel()

	certs := []*iam.ServerCertificateMetadata{
		{
			ServerCertificateName: aws.String("oldest"),
			Expiration:            aws.Time(time.Now()),
		},
		{
			ServerCertificateName: aws.String("latest"),
			Expiration:            aws.Time(time.Now().Add(3 * time.Hour)),
		},
		{
			ServerCertificateName: aws.String("in between"),
			Expiration:            aws.Time(time.Now().Add(2 * time.Hour)),
		},
	}
	sort.Sort(tfiam.CertificateByExpiration(certs))
	if aws.StringValue(certs[0].ServerCertificateName) != "latest" {
		t.Fatalf("Expected first item to be %q, but was %q", "latest", *certs[0].ServerCertificateName)
	}
}

func TestAccIAMServerCertificateDataSource_basic(t *testing.T) {
	ctx := acctest.Context(t)
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	key := acctest.TLSRSAPrivateKeyPEM(t, 2048)
	certificate := acctest.TLSRSAX509SelfSignedCertificatePEM(t, key, "example.com")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, iam.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckServerCertificateDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccServerCertificateDataSourceConfig_cert(rName, key, certificate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("aws_iam_server_certificate.test_cert", "arn"),
					resource.TestCheckResourceAttrSet("data.aws_iam_server_certificate.test", "arn"),
					resource.TestCheckResourceAttrSet("data.aws_iam_server_certificate.test", "id"),
					resource.TestCheckResourceAttrSet("data.aws_iam_server_certificate.test", "name"),
					resource.TestCheckResourceAttrSet("data.aws_iam_server_certificate.test", "path"),
					resource.TestCheckResourceAttrSet("data.aws_iam_server_certificate.test", "upload_date"),
					resource.TestCheckResourceAttr("data.aws_iam_server_certificate.test", "certificate_chain", ""),
					resource.TestMatchResourceAttr("data.aws_iam_server_certificate.test", "certificate_body", regexp.MustCompile("^-----BEGIN CERTIFICATE-----")),
				),
			},
		},
	})
}

func TestAccIAMServerCertificateDataSource_matchNamePrefix(t *testing.T) {
	ctx := acctest.Context(t)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, iam.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckServerCertificateDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config:      testAccServerCertificateDataSourceConfig_certMatchNamePrefix,
				ExpectError: regexp.MustCompile(`Search for AWS IAM server certificate returned no results`),
			},
		},
	})
}

func TestAccIAMServerCertificateDataSource_path(t *testing.T) {
	ctx := acctest.Context(t)
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	path := "/test-path/"
	pathPrefix := "/test-path/"

	key := acctest.TLSRSAPrivateKeyPEM(t, 2048)
	certificate := acctest.TLSRSAX509SelfSignedCertificatePEM(t, key, "example.com")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, iam.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckServerCertificateDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccServerCertificateDataSourceConfig_certPath(rName, path, pathPrefix, key, certificate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.aws_iam_server_certificate.test", "path", path),
				),
			},
		},
	})
}

func testAccServerCertificateDataSourceConfig_cert(rName, key, certificate string) string {
	return fmt.Sprintf(`
resource "aws_iam_server_certificate" "test_cert" {
  name             = "%[1]s"
  certificate_body = "%[2]s"
  private_key      = "%[3]s"
}

data "aws_iam_server_certificate" "test" {
  name   = aws_iam_server_certificate.test_cert.name
  latest = true
}
`, rName, acctest.TLSPEMEscapeNewlines(certificate), acctest.TLSPEMEscapeNewlines(key))
}

func testAccServerCertificateDataSourceConfig_certPath(rName, path, pathPrefix, key, certificate string) string {
	return fmt.Sprintf(`
resource "aws_iam_server_certificate" "test_cert" {
  name             = "%[1]s"
  path             = "%[2]s"
  certificate_body = "%[3]s"
  private_key      = "%[4]s"
}

data "aws_iam_server_certificate" "test" {
  name        = aws_iam_server_certificate.test_cert.name
  path_prefix = "%[5]s"
  latest      = true
}
`, rName, path, acctest.TLSPEMEscapeNewlines(certificate), acctest.TLSPEMEscapeNewlines(key), pathPrefix)
}

var testAccServerCertificateDataSourceConfig_certMatchNamePrefix = `
data "aws_iam_server_certificate" "test" {
  name_prefix = "MyCert"
  latest      = true
}
`
