package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func validCustomerCPF() *Customer {
	return &Customer{
		Name:         "João Silva",
		Email:        "joao@example.com",
		Document:     "111.444.777-35",
		DocumentType: DocumentTypeCPF,
	}
}

func validCustomerCNPJ() *Customer {
	return &Customer{
		Name:         "Empresa LTDA",
		Email:        "empresa@example.com",
		Document:     "11.222.333/0001-81",
		DocumentType: DocumentTypeCNPJ,
	}
}

func TestValidateDocument_ValidCPF(t *testing.T) {
	c := validCustomerCPF()
	assert.NoError(t, c.ValidateDocument())
}

func TestValidateDocument_ValidCNPJ(t *testing.T) {
	c := validCustomerCNPJ()
	assert.NoError(t, c.ValidateDocument())
}

func TestValidateDocument_NameRequired(t *testing.T) {
	c := validCustomerCPF()
	c.Name = ""
	assert.ErrorIs(t, c.ValidateDocument(), ErrCustomerNameRequired)
}

func TestValidateDocument_EmailRequired(t *testing.T) {
	c := validCustomerCPF()
	c.Email = ""
	assert.ErrorIs(t, c.ValidateDocument(), ErrCustomerEmailRequired)
}

func TestValidateDocument_InvalidEmailFormat(t *testing.T) {
	c := validCustomerCPF()
	c.Email = "not-an-email"
	assert.ErrorIs(t, c.ValidateDocument(), ErrCustomerInvalidEmailFormat)
}

func TestValidateDocument_InvalidDocumentType(t *testing.T) {
	c := validCustomerCPF()
	c.DocumentType = "RG"
	assert.ErrorIs(t, c.ValidateDocument(), ErrCustomerInvalidDocumentType)
}

func TestValidateDocument_CPFAllSameDigits(t *testing.T) {
	c := validCustomerCPF()
	c.Document = "111.111.111-11"
	assert.ErrorIs(t, c.ValidateDocument(), ErrCustomerInvalidCPFChecksum)
}

func TestValidateDocument_CPFTooShort(t *testing.T) {
	c := validCustomerCPF()
	c.Document = "123.456"
	assert.ErrorIs(t, c.ValidateDocument(), ErrCustomerInvalidCPFFormat)
}

func TestValidateDocument_CPFInvalidChecksum(t *testing.T) {
	c := validCustomerCPF()
	c.Document = "111.444.777-00"
	assert.ErrorIs(t, c.ValidateDocument(), ErrCustomerInvalidCPFChecksum)
}

func TestValidateDocument_CNPJTooShort(t *testing.T) {
	c := validCustomerCNPJ()
	c.Document = "11.222.333"
	assert.ErrorIs(t, c.ValidateDocument(), ErrCustomerInvalidCNPJFormat)
}

func TestValidateDocument_CNPJInvalidChecksum(t *testing.T) {
	c := validCustomerCNPJ()
	c.Document = "11.222.333/0001-00"
	assert.ErrorIs(t, c.ValidateDocument(), ErrCustomerInvalidCNPJChecksum)
}

func TestNormalize_StripsDocumentMask(t *testing.T) {
	c := &Customer{
		Name:     "  João  ",
		Email:    "  joao@example.com  ",
		Document: "111.444.777-35",
	}
	c.Normalize()
	assert.Equal(t, "João", c.Name)
	assert.Equal(t, "joao@example.com", c.Email)
	assert.Equal(t, "11144477735", c.Document)
}

func TestNormalize_CNPJStripsAllSeparators(t *testing.T) {
	c := &Customer{Document: "11.222.333/0001-81"}
	c.Normalize()
	assert.Equal(t, "11222333000181", c.Document)
}
