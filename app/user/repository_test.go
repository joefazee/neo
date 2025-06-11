package user

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"

	"github.com/joefazee/neo/models"
	"github.com/joefazee/neo/tests/suites"
)

type UserRepositoryTestSuite struct {
	suites.RepositoryTestSuite
	repo Repository
}

func (suite *UserRepositoryTestSuite) SetupSuite() {
	if testing.Short() {
		suite.T().Skip("Skipping database integration test")
	}

	suite.AutoMigrate = true
	suite.RepositoryTestSuite.SetupSuite()
	suite.repo = NewRepository(suite.DB)
}

func TestUserRepository(t *testing.T) {
	suite.Run(t, new(UserRepositoryTestSuite))
}

func (suite *UserRepositoryTestSuite) TestCreate() {
	user := suite.createTestUser("test@example.com", "+1234567890")

	assert.NotEqual(suite.T(), uuid.Nil, user.ID)
	assert.Equal(suite.T(), "test@example.com", user.Email)
}

func (suite *UserRepositoryTestSuite) TestGetByEmail() {
	ctx := context.Background()
	email := "getby@example.com"
	suite.createTestUser(email, "+1234567891")

	user, err := suite.repo.GetByEmail(ctx, email)
	suite.AssertNoDBError(err)
	suite.Assert().Equal(email, user.Email)
}

func (suite *UserRepositoryTestSuite) TestGetByEmail_NotFound() {
	ctx := context.Background()

	user, err := suite.repo.GetByEmail(ctx, "notfound@example.com")
	suite.AssertDBError(err)
	suite.Assert().Nil(user)
	suite.Assert().ErrorIs(err, models.ErrRecordNotFound)
}

func (suite *UserRepositoryTestSuite) TestGetByPhone() {
	ctx := context.Background()
	phone := "+1234567892"
	suite.createTestUser("phone@example.com", phone)

	user, err := suite.repo.GetByPhone(ctx, phone)
	suite.AssertNoDBError(err)
	suite.Assert().Equal(phone, user.Phone)
}

func (suite *UserRepositoryTestSuite) TestGetByPhone_NotFound() {
	ctx := context.Background()

	user, err := suite.repo.GetByPhone(ctx, "+9999999999")
	suite.AssertDBError(err)
	suite.Assert().Nil(user)
	suite.Assert().ErrorIs(err, models.ErrRecordNotFound)
}

func (suite *UserRepositoryTestSuite) TestUpdate() {
	ctx := context.Background()
	user := suite.createTestUser("update@example.com", "+1234567893")

	user.FirstName = "Updated"
	user.LastName = "Name"
	*user.IsActive = false

	err := suite.repo.Update(ctx, user)
	suite.AssertNoDBError(err)

	updated, err := suite.repo.GetByID(ctx, user.ID)
	suite.AssertNoDBError(err)
	suite.Assert().Equal("Updated", updated.FirstName)
	suite.Assert().Equal("Name", updated.LastName)
	suite.Assert().False(*updated.IsActive)
}

func (suite *UserRepositoryTestSuite) TestGetByID() {
	ctx := context.Background()
	user := suite.createTestUser("getbyid@example.com", "+1234567894")

	found, err := suite.repo.GetByID(ctx, user.ID)
	suite.AssertNoDBError(err)
	suite.Assert().Equal(user.ID, found.ID)
	suite.Assert().Equal(user.Email, found.Email)
}

func (suite *UserRepositoryTestSuite) TestGetByID_NotFound() {
	ctx := context.Background()

	_, err := suite.repo.GetByID(ctx, uuid.New())
	suite.AssertDBError(err)
	suite.Assert().ErrorIs(err, gorm.ErrRecordNotFound)
}

func (suite *UserRepositoryTestSuite) TestGetByIDWithPermissions() {
	ctx := context.Background()
	user := suite.createTestUser("perms@example.com", "+1234567895")
	role := suite.createTestRole("test_role")
	permission := suite.createTestPermission("test_permission")

	suite.assignPermissionToRole(ctx, role.ID, permission.ID)
	suite.assignRoleToUser(user.ID, role.ID)

	found, err := suite.repo.GetByIDWithPermissions(ctx, user.ID)
	suite.AssertNoDBError(err)
	suite.Assert().Len(found.Roles, 1)
	suite.Assert().Len(found.Roles[0].Permissions, 1)
	suite.Assert().Equal("test_permission", found.Roles[0].Permissions[0].Name)
}

func (suite *UserRepositoryTestSuite) TestGetUsers_NoFilters() {
	ctx := context.Background()
	suite.createTestUser("johne@example.com", "+1111111111")
	suite.createTestUser("jane@example.com", "+2222222222")

	filters := &AdminUserFilters{Page: 1, PerPage: 10}
	users, total, err := suite.repo.GetUsers(ctx, filters)

	suite.AssertNoDBError(err)
	suite.Assert().GreaterOrEqual(int(total), 2)
	suite.Assert().GreaterOrEqual(len(users), 2)
}

func (suite *UserRepositoryTestSuite) TestGetUsers_StatusFilter() {
	ctx := context.Background()
	activeUser := suite.createTestUserWithStatus("active@example.com", "+3333333333", true)
	inactiveUser := suite.createTestUserWithStatus("inactive@example.com", "+4444444444", false)

	suite.Assert().NotNil(activeUser)
	suite.Assert().NotNil(inactiveUser)

	filters := &AdminUserFilters{Page: 1, PerPage: 10, Status: "active"}
	users, total, err := suite.repo.GetUsers(ctx, filters)

	suite.AssertNoDBError(err)
	suite.Assert().GreaterOrEqual(int(total), 1)

	for i := range users {
		suite.Assert().True(*users[i].IsActive)
	}

	filters.Status = "inactive"
	users, _, err = suite.repo.GetUsers(ctx, filters)
	suite.AssertNoDBError(err)

	for i := range users {
		suite.Assert().False(*users[i].IsActive)
	}
}

func (suite *UserRepositoryTestSuite) TestGetUsers_SearchFilter() {
	ctx := context.Background()
	user1 := suite.createTestUserWithName("search@example.com", "+5555555555", "John", "Doe")
	suite.Assert().NotNil(user1)
	user2 := suite.createTestUserWithName("other@example.com", "+6666666666", "Jane", "Smith")
	suite.Assert().NotNil(user2)

	filters := &AdminUserFilters{Page: 1, PerPage: 10, Search: "john"}
	users, total, err := suite.repo.GetUsers(ctx, filters)

	suite.AssertNoDBError(err)
	suite.Assert().GreaterOrEqual(int(total), 1)

	found := false
	for i := range users {
		user := users[i]
		if user.FirstName == "John" {
			found = true
			break
		}
	}
	suite.Assert().True(found)
}

func (suite *UserRepositoryTestSuite) TestGetUsers_Sorting() {
	ctx := context.Background()
	suite.createTestUserWithName("a@example.com", "+7777777777", "Alice", "Alpha")
	suite.createTestUserWithName("b@example.com", "+8888888888", "Bob", "Beta")

	filters := &AdminUserFilters{Page: 1, PerPage: 10, SortBy: "first_name", SortOrder: "asc"}
	users, _, err := suite.repo.GetUsers(ctx, filters)

	suite.AssertNoDBError(err)
	suite.Assert().GreaterOrEqual(len(users), 2)
}

func (suite *UserRepositoryTestSuite) TestGetUsers_Pagination() {
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		suite.createTestUser(fmt.Sprintf("%dpage@example.com", i), fmt.Sprintf("+999999999%d", i))
	}

	filters := &AdminUserFilters{Page: 1, PerPage: 2}
	users, total, err := suite.repo.GetUsers(ctx, filters)

	suite.AssertNoDBError(err)
	suite.Assert().GreaterOrEqual(int(total), 5)
	suite.Assert().LessOrEqual(len(users), 2)
}

func (suite *UserRepositoryTestSuite) TestUpdateUserStatus() {
	ctx := context.Background()
	user := suite.createTestUserWithStatus("status@example.com", "+1010101010", true)

	err := suite.repo.UpdateUserStatus(ctx, user.ID, false)
	suite.AssertNoDBError(err)

	updated, err := suite.repo.GetByID(ctx, user.ID)
	suite.AssertNoDBError(err)
	suite.Assert().False(*updated.IsActive)
}

func (suite *UserRepositoryTestSuite) TestAssignRole() {
	ctx := context.Background()
	user := suite.createTestUser("role@example.com", "+1212121212")
	role1 := suite.createTestRole("role1")
	role2 := suite.createTestRole("role2")

	// Assign first role
	err := suite.repo.AssignRole(ctx, user.ID, role1.ID)
	suite.AssertNoDBError(err)

	// Assign second role (should replace first)
	err = suite.repo.AssignRole(ctx, user.ID, role2.ID)
	suite.AssertNoDBError(err)

	// Verify only role2 is assigned
	userWithRoles, err := suite.repo.GetUserByIDWithRoles(ctx, user.ID)
	suite.AssertNoDBError(err)
	suite.Assert().Len(userWithRoles.Roles, 1)
	suite.Assert().Equal(role2.ID, userWithRoles.Roles[0].ID)
}

func (suite *UserRepositoryTestSuite) TestBulkAssignPermissions() {
	ctx := context.Background()
	user1 := suite.createTestUser("bulk1@example.com", "+1313131313")
	user2 := suite.createTestUser("xxxx@example.com", "+1414141414")
	perm1 := suite.createTestPermission("bulk_perm1")
	perm2 := suite.createTestPermission("bulk_perm2")

	userIDs := []uuid.UUID{user1.ID, user2.ID}
	permIDs := []uuid.UUID{perm1.ID, perm2.ID}

	err := suite.repo.BulkAssignPermissions(ctx, userIDs, permIDs)
	suite.AssertDBError(err)
}

func (suite *UserRepositoryTestSuite) TestBulkAssignPermissions_EmptyParams() {
	ctx := context.Background()

	err := suite.repo.BulkAssignPermissions(ctx, []uuid.UUID{}, []uuid.UUID{})
	suite.AssertNoDBError(err) // Should not error on empty params
}

func (suite *UserRepositoryTestSuite) TestBulkAssignPermissions_InvalidUsers() {
	ctx := context.Background()
	perm := suite.createTestPermission("invalid_user_perm")

	err := suite.repo.BulkAssignPermissions(ctx, []uuid.UUID{uuid.New()}, []uuid.UUID{perm.ID})
	suite.AssertDBError(err) // Should error when users don't exist
}

func (suite *UserRepositoryTestSuite) TestBulkAssignPermissions_InvalidPermissions() {
	ctx := context.Background()
	user := suite.createTestUser("invalid_perm@example.com", "+1515151515")

	err := suite.repo.BulkAssignPermissions(ctx, []uuid.UUID{user.ID}, []uuid.UUID{uuid.New()})
	suite.AssertDBError(err) // Should error when no valid permissions found
}

func (suite *UserRepositoryTestSuite) TestCreatePermission() {
	ctx := context.Background()
	permission := &models.Permission{
		Name:        "test_create_permission",
		Description: "Test Permission",
	}

	err := suite.repo.CreatePermission(ctx, permission)
	suite.AssertNoDBError(err)
	suite.Assert().NotEqual(uuid.Nil, permission.ID)
}

func (suite *UserRepositoryTestSuite) TestGetPermissionByName() {
	ctx := context.Background()
	created := suite.createTestPermission("get_by_name_perm")

	found, err := suite.repo.GetPermissionByName(ctx, "get_by_name_perm")
	suite.AssertNoDBError(err)
	suite.Assert().Equal(created.ID, found.ID)
}

func (suite *UserRepositoryTestSuite) TestGetPermissionByName_NotFound() {
	ctx := context.Background()

	_, err := suite.repo.GetPermissionByName(ctx, "nonexistent_permission")
	suite.AssertDBError(err)
	suite.Assert().ErrorIs(err, gorm.ErrRecordNotFound)
}

func (suite *UserRepositoryTestSuite) TestGetPermissionsByNames() {
	ctx := context.Background()
	perm1 := suite.createTestPermission("multi_perm1")
	perm2 := suite.createTestPermission("multi_perm2")
	suite.Assert().NotNil(perm1)
	suite.Assert().NotNil(perm2)

	names := []string{"multi_perm1", "multi_perm2", "nonexistent"}
	permissions, err := suite.repo.GetPermissionsByNames(ctx, names)

	suite.AssertNoDBError(err)
	suite.Assert().Len(permissions, 2)

	foundNames := make(map[string]bool)
	for _, perm := range permissions {
		foundNames[perm.Name] = true
	}
	suite.Assert().True(foundNames["multi_perm1"])
	suite.Assert().True(foundNames["multi_perm2"])
}

func (suite *UserRepositoryTestSuite) TestCreateRole() {
	ctx := context.Background()
	role := &models.Role{
		Name:        "test_create_role",
		Description: "Test Role",
	}

	err := suite.repo.CreateRole(ctx, role)
	suite.AssertNoDBError(err)
	suite.Assert().NotEqual(uuid.Nil, role.ID)
}

func (suite *UserRepositoryTestSuite) TestUpdateRole() {
	ctx := context.Background()
	role := suite.createTestRole("update_role")

	role.Name = "updated_role"
	role.Description = "Updated Description"

	err := suite.repo.UpdateRole(ctx, role)
	suite.AssertNoDBError(err)

	updated, err := suite.repo.GetRoleByID(ctx, role.ID)
	suite.AssertNoDBError(err)
	suite.Assert().Equal("updated_role", updated.Name)
	suite.Assert().Equal("Updated Description", updated.Description)
}

func (suite *UserRepositoryTestSuite) TestGetRoleByID() {
	ctx := context.Background()
	role := suite.createTestRole("get_role_by_id")
	permission := suite.createTestPermission("role_permission")
	suite.assignPermissionToRole(ctx, role.ID, permission.ID)

	found, err := suite.repo.GetRoleByID(ctx, role.ID)
	suite.AssertNoDBError(err)
	suite.Assert().Equal(role.ID, found.ID)
	suite.Assert().Len(found.Permissions, 1)
	suite.Assert().Equal("role_permission", found.Permissions[0].Name)
}

func (suite *UserRepositoryTestSuite) TestGetRoleByID_NotFound() {
	ctx := context.Background()

	_, err := suite.repo.GetRoleByID(ctx, uuid.New())
	suite.AssertDBError(err)
	suite.Assert().ErrorIs(err, gorm.ErrRecordNotFound)
}

func (suite *UserRepositoryTestSuite) TestAssignPermissionsToRole() {
	ctx := context.Background()
	role := suite.createTestRole("assign_perms_role")
	perm1 := suite.createTestPermission("assign_perm1")
	perm2 := suite.createTestPermission("assign_perm2")

	permIDs := []uuid.UUID{perm1.ID, perm2.ID}
	err := suite.repo.AssignPermissionsToRole(ctx, role.ID, permIDs)
	suite.AssertNoDBError(err, "Failed to assign permissions to role")

	updated, err := suite.repo.GetRoleByID(ctx, role.ID)
	suite.AssertNoDBError(err)
	suite.Assert().Len(updated.Permissions, 2)
}

func (suite *UserRepositoryTestSuite) TestAssignPermissionsToRole_Errors() {
	ctx := context.Background()

	nonExistentRoleID := uuid.New()
	err := suite.repo.AssignPermissionsToRole(ctx, nonExistentRoleID, []uuid.UUID{})
	suite.AssertDBError(err)
	suite.Assert().Equal(err.Error(), fmt.Sprintf("role %s does not exist", nonExistentRoleID))

	role := suite.createTestRole("assign_perms_error_role")
	perm := suite.createTestPermission("assign_perm_error")
	err = suite.repo.AssignPermissionsToRole(ctx, role.ID, []uuid.UUID{perm.ID, uuid.New()})
	suite.Assert().NotNil(err)
	suite.Assert().Equal(err.Error(), "some permissions do not exist")
}

func (suite *UserRepositoryTestSuite) TestAssignPermissionsToRole_RoleNotFound() {
	ctx := context.Background()
	perm := suite.createTestPermission("invalid_role_perm")

	nonExistentRoleID := uuid.New()

	err := suite.repo.AssignPermissionsToRole(ctx, nonExistentRoleID, []uuid.UUID{perm.ID})
	suite.AssertDBError(err)
	suite.Assert().Equal(err.Error(), fmt.Sprintf("role %s does not exist", nonExistentRoleID))
}

func (suite *UserRepositoryTestSuite) TestRemovePermissionsFromRole() {
	ctx := context.Background()
	role := suite.createTestRole("remove_perms_role")
	perm1 := suite.createTestPermission("remove_perm1")
	perm2 := suite.createTestPermission("remove_perm2")

	// Assign permissions first
	permIDs := []uuid.UUID{perm1.ID, perm2.ID}
	err := suite.repo.AssignPermissionsToRole(ctx, role.ID, permIDs)
	suite.AssertNoDBError(err)

	// Remove one permission
	err = suite.repo.RemovePermissionsFromRole(ctx, role.ID, []uuid.UUID{perm1.ID})
	suite.AssertNoDBError(err)

	updated, err := suite.repo.GetRoleByID(ctx, role.ID)
	suite.AssertNoDBError(err)
	suite.Assert().Len(updated.Permissions, 1)
	suite.Assert().Equal(perm2.ID, updated.Permissions[0].ID)
}

func (suite *UserRepositoryTestSuite) TestGetUserByIDWithRoles() {
	ctx := context.Background()
	country := suite.createTestCountry("TC 1", "TC")
	user := suite.createTestUserWithCountry("roles@example.com", "+1616161616", country.ID)
	role := suite.createTestRole("user_role")
	permission := suite.createTestPermission("user_permission")

	suite.assignPermissionToRole(ctx, role.ID, permission.ID)
	suite.assignRoleToUser(user.ID, role.ID)

	found, err := suite.repo.GetUserByIDWithRoles(ctx, user.ID)
	suite.AssertNoDBError(err)
	suite.Assert().Equal(user.ID, found.ID)
	suite.Assert().Len(found.Roles, 1)
	suite.Assert().Len(found.Roles[0].Permissions, 1)
	suite.Assert().NotNil(found.Country)
	suite.Assert().Equal(country.ID, found.Country.ID)

	found, err = suite.repo.GetUserByIDWithRoles(ctx, uuid.New())
	suite.AssertDBError(err)
	suite.Assert().Nil(found, "Expected nil user when ID not found")
}

func (suite *UserRepositoryTestSuite) TestRemoveRoleFromUser() {
	ctx := context.Background()
	user := suite.createTestUser("remove_role@example.com", "+1717171717")
	role := suite.createTestRole("remove_user_role")

	suite.assignRoleToUser(user.ID, role.ID)

	err := suite.repo.RemoveRoleFromUser(ctx, user.ID, role.ID)
	suite.AssertNoDBError(err)

	userWithRoles, err := suite.repo.GetUserByIDWithRoles(ctx, user.ID)
	suite.AssertNoDBError(err)
	suite.Assert().Empty(userWithRoles.Roles)
}

func (suite *UserRepositoryTestSuite) TestRemoveRoleFromUser_UserNotFound() {
	ctx := context.Background()
	role := suite.createTestRole("invalid_user_role")

	err := suite.repo.RemoveRoleFromUser(ctx, uuid.New(), role.ID)
	suite.AssertDBError(err)
	suite.Assert().ErrorIs(err, gorm.ErrRecordNotFound)
}

func (suite *UserRepositoryTestSuite) TestRemoveRoleFromUser_RoleNotFound() {
	ctx := context.Background()
	user := suite.createTestUser("invalid_role@example.com", "+1818181818")

	err := suite.repo.RemoveRoleFromUser(ctx, user.ID, uuid.New())
	suite.AssertDBError(err)
	suite.Assert().ErrorIs(err, gorm.ErrRecordNotFound)
}

// Helper methods

func (suite *UserRepositoryTestSuite) createTestUser(email, phone string) *models.User {
	country := suite.createTestCountry(email, email[:2])
	return suite.createTestUserWithCountry(email, phone, country.ID)
}

func (suite *UserRepositoryTestSuite) createTestUserWithStatus(email, phone string, isActive bool) *models.User {
	country := suite.createTestCountry(email, email[:2])
	user := &models.User{
		CountryID:    country.ID,
		Email:        email,
		FirstName:    "Test",
		LastName:     "User",
		Phone:        phone,
		PasswordHash: "123456pasw",
		IsActive:     &isActive,
	}
	err := suite.repo.Create(context.Background(), user)
	suite.AssertNoDBError(err)
	return user
}

func (suite *UserRepositoryTestSuite) createTestUserWithName(email, phone, firstName, lastName string) *models.User {
	country := suite.createTestCountry(email, email[:2])
	isActive := true
	user := &models.User{
		CountryID:    country.ID,
		Email:        email,
		FirstName:    firstName,
		LastName:     lastName,
		Phone:        phone,
		PasswordHash: "$2a$10$test",
		IsActive:     &isActive,
	}
	err := suite.repo.Create(context.Background(), user)
	suite.AssertNoDBError(err)
	return user
}

func (suite *UserRepositoryTestSuite) createTestUserWithCountry(email, phone string, countryID uuid.UUID) *models.User {
	isActive := true
	user := &models.User{
		CountryID:    countryID,
		Email:        email,
		FirstName:    "Test",
		LastName:     "User",
		Phone:        phone,
		PasswordHash: "$2a$10$test",
		IsActive:     &isActive,
		Metadata: &models.UserMetadata{
			ReferralCode: "XYZ123",
		},
	}
	err := suite.repo.Create(context.Background(), user)
	suite.AssertNoDBError(err)
	return user
}

func (suite *UserRepositoryTestSuite) createTestPermission(name string) *models.Permission {
	permission := &models.Permission{
		Name:        name,
		Description: "Test " + name,
	}
	err := suite.repo.CreatePermission(context.Background(), permission)
	suite.AssertNoDBError(err)
	return permission
}

func (suite *UserRepositoryTestSuite) createTestRole(name string) *models.Role {
	role := &models.Role{
		Name:        name,
		Description: "Test " + name,
	}
	err := suite.repo.CreateRole(context.Background(), role)
	suite.AssertNoDBError(err)
	return role
}

func (suite *UserRepositoryTestSuite) createTestCountry(name, code string) *models.Country {
	isActive := true
	country := &models.Country{
		Name:           name,
		Code:           code,
		CurrencyCode:   "USD",
		CurrencySymbol: "$",
		IsActive:       &isActive,
	}
	err := suite.DB.Create(country).Error
	suite.AssertNoDBError(err)
	return country
}

func (suite *UserRepositoryTestSuite) assignPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) {
	err := suite.repo.AssignPermissionsToRole(ctx, roleID, []uuid.UUID{permissionID})
	suite.AssertNoDBError(err)
}

func (suite *UserRepositoryTestSuite) assignRoleToUser(userID, roleID uuid.UUID) {
	err := suite.DB.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", userID, roleID).Error
	suite.AssertNoDBError(err)
}
