package api

import (
	"github.com/icodeologist/atomicurl/internal/models"
	"github.com/icodeologist/atomicurl/internal/utils"
)

// These aliases keep HTTP handlers focused on transport concerns while the
// domain models and reusable URL helpers live in their own internal packages.
type User = models.User
type Url = models.Url
type Link = models.Link
type LinkVersion = models.LinkVersion

var ValidateAndNormalizeURL = utils.ValidateAndNormalizeURL
var GenerateShortIDWithBase62Encoding = utils.GenerateShortIDWithBase62Encoding
