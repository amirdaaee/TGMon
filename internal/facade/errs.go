package facade

import "errors"

var ErrFileAlreadyExists = errors.New("file already exists")
var ErrNoDocumentsFound = errors.New("no documents found")
var ErrMultipleDocumentsFound = errors.New("multiple documents found")
