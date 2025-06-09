package user

import (
	"context"
	"strings"
	"time"

	"github.com/joefazee/neo/app/countries"

	"github.com/joefazee/neo/internal/formatter"
	"github.com/joefazee/neo/internal/sanitizer"

	"github.com/joefazee/neo/internal/validator"

	"github.com/google/uuid"
)

// RegisterUserRequest represents the request to create a user.
type RegisterUserRequest struct {
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	Email       string    `json:"email"`
	CountryCode string    `json:"country_code"`
	PhoneNumber string    `json:"phone_number"`
	Password    string    `json:"password"`
	CountryID   uuid.UUID `json:"-"`
}

func (r *RegisterUserRequest) Validate(ctx context.Context,
	v *validator.Validator,
	countryRepo countries.Repository, s sanitizer.HTMLStripperer) bool {
	r.FirstName = s.StripHTML(r.FirstName)
	r.LastName = s.StripHTML(r.LastName)
	r.Email = s.StripHTML(r.Email)
	r.PhoneNumber = s.StripHTML(r.PhoneNumber)

	v.Check(r.FirstName != "", "first_name", "first name is required")
	v.Check(r.LastName != "", "last_name", "last name is required")
	v.Check(validator.MinRunes(r.FirstName, 2) && validator.MaxRunes(r.FirstName, 150), "first_name", "first name must be between 2 and 150 characters")
	v.Check(validator.MinRunes(r.LastName, 2) && validator.MaxRunes(r.LastName, 150), "last_name", "last name must be between 2 and 150 characters")
	v.Check(validator.IsEmail(r.Email), "email", "email is invalid")
	v.Check(r.CountryCode != "", "country_code", "country code is required")

	r.Email = strings.ToLower(r.Email)
	country, err := countryRepo.GetByCode(ctx, r.CountryCode)
	if err != nil {
		v.AddError("country_code", "invalid country code")
		return v.Valid()
	}

	v.Check(country != nil, "country_code", "country code does not exist")
	v.Check(*country.IsActive, "country_code", "country is not active")
	r.CountryID = country.ID

	r.PhoneNumber, err = formatter.FormatPhone(r.PhoneNumber, r.CountryCode)
	if err != nil {
		v.AddError("phone_number", "invalid phone number")
		return v.Valid()
	}
	v.Check(r.PhoneNumber != "", "phone_number", "phone number is required")

	return v.Valid()
}

// LoginRequest represents the request to log in.
type LoginRequest struct {
	Identity    string `json:"identity"`
	Password    string `json:"password"`
	CountryCode string `json:"country_code"`
}

func (r *LoginRequest) Validate(ctx context.Context,
	v *validator.Validator,
	countryRepo countries.Repository, s sanitizer.HTMLStripperer) bool {
	r.Identity = s.StripHTML(r.Identity)

	v.Check(r.Identity != "", "identity", "identity is required")
	v.Check(r.Password != "", "password", "password is required")
	v.Check(validator.MinRunes(r.Password, 8), "password", "password must be at least 8 characters long")
	v.Check(validator.MinRunes(r.Password, 8), "password", "password must be at least 8 characters long")

	if r.CountryCode != "" {
		r.CountryCode = s.StripHTML(r.CountryCode)
		v.Check(r.CountryCode != "", "country_code", "country code is required")

		country, err := countryRepo.GetByCode(ctx, r.CountryCode)
		if err != nil {
			v.AddError("country_code", "invalid country code")
			return v.Valid()
		}

		v.Check(country != nil, "country_code", "country code does not exist")
		v.Check(*country.IsActive, "country_code", "country is not active")

		r.Identity, _ = formatter.FormatPhone(r.Identity, r.CountryCode)
	}

	return v.Valid()
}

// PasswordResetRequest represents the request to initiate a password reset.
type PasswordResetRequest struct {
	Email string `json:"email"`
}

// SetNewPasswordRequest represents the request to set a new password.
type SetNewPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// Response represents the response for user data.
type Response struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email,omitempty"`
	Phone     string    `json:"phone,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// LoginResponse represents the response for a successful login.
type LoginResponse struct {
	AccessToken string   `json:"access_token"`
	User        Response `json:"user"`
}
