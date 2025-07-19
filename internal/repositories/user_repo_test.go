package repositories

import (
	"regexp"

	"ggcode/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupMockDB() (*gorm.DB, sqlmock.Sqlmock, func()) {
	mockDB, mock, err := sqlmock.New()
	Expect(err).NotTo(HaveOccurred())
	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	Expect(err).NotTo(HaveOccurred())
	cleanup := func() { mockDB.Close() }
	return db, mock, cleanup
}

var _ = Describe("UserRepository", func() {
	var (
		db      *gorm.DB
		mock    sqlmock.Sqlmock
		cleanup func()
		repo    UserRepository
	)

	BeforeEach(func() {
		db, mock, cleanup = setupMockDB()
		repo = NewUserRepository(db)
	})

	AfterEach(func() {
		cleanup()
	})

	Describe("Create", func() {
		It("should create user successfully", func() {
			user := &models.User{Username: "testuser", Email: "test@example.com"}
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users"`)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
			mock.ExpectCommit()

			err := repo.Create(user)
			Expect(err).NotTo(HaveOccurred())
			Expect(mock.ExpectationsWereMet()).To(Succeed())
		})
	})

	Describe("GetByUsername", func() {
		It("should return user when found", func() {
			user := &models.User{ID: 1, Username: "testuser", Email: "test@example.com"}
			rows := sqlmock.NewRows([]string{"id", "username", "email"}).AddRow(user.ID, user.Username, user.Email)
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE username = $1 ORDER BY "users"."id" LIMIT $2`)).WithArgs("testuser", 1).WillReturnRows(rows)

			result, err := repo.GetByUsername("testuser")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Username).To(Equal(user.Username))
			Expect(mock.ExpectationsWereMet()).To(Succeed())
		})

		It("should return error when user not found", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE username = $1 ORDER BY "users"."id" LIMIT $2`)).WithArgs("nouser", 1).WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email"}))
			result, err := repo.GetByUsername("nouser")
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Describe("GetByUsernameOrEmail", func() {
		It("should return user when found", func() {
			user := &models.User{ID: 2, Username: "user2", Email: "user2@example.com"}
			rows := sqlmock.NewRows([]string{"id", "username", "email"}).AddRow(user.ID, user.Username, user.Email)
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE username = $1 OR email = $2 ORDER BY "users"."id" LIMIT $3`)).WithArgs("user2", "user2@example.com", 1).WillReturnRows(rows)

			result, err := repo.GetByUsernameOrEmail("user2", "user2@example.com")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Email).To(Equal(user.Email))
			Expect(mock.ExpectationsWereMet()).To(Succeed())
		})

		It("should return error when user not found", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE username = $1 OR email = $2 ORDER BY "users"."id" LIMIT $3`)).WithArgs("nouser", "noemail", 1).WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email"}))
			result, err := repo.GetByUsernameOrEmail("nouser", "noemail")
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Describe("IsAdmin", func() {
		It("should return true for admin user", func() {
			rows := sqlmock.NewRows([]string{"is_admin"}).AddRow(true)
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT "is_admin" FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).WithArgs(1, 1).WillReturnRows(rows)

			isAdmin, err := repo.IsAdmin(1)
			Expect(err).NotTo(HaveOccurred())
			Expect(isAdmin).To(BeTrue())
			Expect(mock.ExpectationsWereMet()).To(Succeed())
		})

		It("should return false for non-admin user", func() {
			rows := sqlmock.NewRows([]string{"is_admin"}).AddRow(false)
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT "is_admin" FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).WithArgs(2, 1).WillReturnRows(rows)

			isAdmin, err := repo.IsAdmin(2)
			Expect(err).NotTo(HaveOccurred())
			Expect(isAdmin).To(BeFalse())
			Expect(mock.ExpectationsWereMet()).To(Succeed())
		})

		It("should return error when user not found", func() {
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT "is_admin" FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).WithArgs(999, 1).WillReturnError(gorm.ErrRecordNotFound)
			isAdmin, err := repo.IsAdmin(999)
			Expect(err).To(HaveOccurred())
			Expect(isAdmin).To(BeFalse())
		})
	})
})
