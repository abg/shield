package db_test

import (
	"fmt"

	. "db"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"
)

func setupORM(sqls ...string) (*ORM, error) {
	var database *DB
	database = &DB{
		Driver: "sqlite3",
		DSN:    ":memory:",
	}

	if err := database.Connect(); err != nil {
		return nil, err
	}

	orm, err := NewORM(database)
	if err != nil {
		database.Disconnect()
		return nil, err
	}

	if err := orm.Setup(); err != nil {
		database.Disconnect()
		return nil, err
	}

	for _, s := range sqls {
		err := database.Exec(s)
		if err != nil {
			database.Disconnect()
			return nil, err
		}
	}

	return orm, nil
}

var _ = Describe("ORM", func() {
	Describe("Initializing the schema", func() {
		Context("With a new database", func() {
			var db *DB

			BeforeEach(func() {
				db = &DB{
					Driver: "sqlite3",
					DSN:    ":memory:",
				}

				Ω(db.Connect()).ShouldNot(HaveOccurred())
				Ω(db.Connected()).Should(BeTrue())
			})

			It("should succeed", func() {
				orm, err := NewORM(db)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(orm).ShouldNot(BeNil())
			})

			It("should not create tables until Setup() is called", func() {
				orm, err := NewORM(db)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(orm).ShouldNot(BeNil())

				Ω(db.Exec("SELECT * FROM schema_info")).
					Should(HaveOccurred())
			})

			It("should create tables during Setup()", func() {
				orm, err := NewORM(db)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(orm).ShouldNot(BeNil())

				Ω(orm.Setup()).ShouldNot(HaveOccurred())

				Ω(db.Exec("SELECT * FROM schema_info")).
					ShouldNot(HaveOccurred())
			})

			It("should set the version number in schema_info", func() {
				orm, err := NewORM(db)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(orm).ShouldNot(BeNil())

				Ω(orm.Setup()).ShouldNot(HaveOccurred())

				r, err := db.Query(`SELECT version FROM schema_info`)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(r).ShouldNot(BeNil())
				Ω(r.Next()).Should(BeTrue())

				var v int
				Ω(r.Scan(&v)).ShouldNot(HaveOccurred())
				Ω(v).Should(Equal(1))
			})

			It("creates the correct tables", func() {
				orm, err := NewORM(db)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(orm).ShouldNot(BeNil())

				Ω(orm.Setup()).ShouldNot(HaveOccurred())

				tableExists := func(table string) {
					sql := fmt.Sprintf("SELECT * FROM %s", table)
					Ω(db.Exec(sql)).ShouldNot(HaveOccurred())
				}

				tableExists("targets")
				tableExists("stores")
				tableExists("schedules")
				tableExists("retention")
				tableExists("jobs")
				tableExists("archives")
				tableExists("tasks")
			})
		})
	})
	Describe("Retrieving Jobs", func() {
		var db *DB
		var orm *ORM

		BeforeEach(func() {
			var err error
			orm, err = setupORM()
			Ω(err).ShouldNot(HaveOccurred())
		})

		Context("With an empty database", func() {
			It("should return an empty list of jobs", func() {
				Ω(orm).ShouldNot(BeNil())
				jobs, err := orm.GetAllJobs()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(len(jobs)).Should(Equal(0))
			})
		})
		/*
		Context("With a non-empty database", func() {
			BeforeEach(func() {
				db.Alias("new-job", `
					INSERT INTO jobs (uuid, target_uuid, store_uuid, schedule_uuid, retention_uuid, paused, name, summary) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
				`)
				db.Alias("new-target", `
					INSERT INTO targets (uuid, name, summary, plugin, endpoint) VALUES (?, ?, ?, ?, ?)
				`)
				db.Alias("new-store", `
					INSERT INTO stores (uuid, name, summary, plugin, endpoint) VALUES (?, ?, ?, ?, ?)
				`)
				db.Alias("new-schedule", `
					INSERT INTO schedules (uuid, name, summary, timespec) VALUES (?, ?, ?, ?)
				`)
				db.Alias("new-retention", `
					INSERT INTO retention (uuid, name, summary, expiry) VALUES (?, ?, ?, ?)
				`)

				// Jobs
				err := db.Exec("new-job",
					"6809c55a-8250-11e5-8bcf-feff819cdc9f", // job UUID
					"c957554e-6fe0-4ae9-816d-307e20f155cb", // target_uuid
					"32d810b6-073b-4296-8c68-6544a91760f9", // store_uuid
					"b36eeea3-9f5c-46f2-a337-6de5344e8d0f", // schedule_uuid
					"52c6f512-5c90-4364-998c-f849fa416243", // retention_uuid
					true,
					"job 1",
					"First test job in queue")
				Ω(err).ShouldNot(HaveOccurred())

				err = db.Exec("new-job",
					"b20ca8b6-8250-11e5-8bcf-feff819cdc9f", // job UUID
					"cf65a73e-79c1-48e8-b706-23ec7644c721", // target_uuid
					"9eb022c4-227f-44f1-b11b-b6d8bcfc3c4f", // store_uuid
					"4b8432b9-b5e9-46d5-b23e-ba70983a2acc", // schedule_uuid
					"f7dee6c2-59c4-439d-9d92-04046b8beb68", // retention_uuid
					false,
					"job 2",
					"Second test job in queue")
				Ω(err).ShouldNot(HaveOccurred())

				// Targets
				err = db.Exec("new-target",
					"c957554e-6fe0-4ae9-816d-307e20f155cb",
					"redis-shared",
					"Shared Redis services for CF",
					"redis",
					`{"host":"10.9.0.23","port":"6379"}`)
				Ω(err).ShouldNot(HaveOccurred())

				err = db.Exec("new-target",
					"cf65a73e-79c1-48e8-b706-23ec7644c721",
					"s3",
					"Amazon S3 Blobstore",
					"s3",
					`{"url":"amazonaws.com","bucket":"bosh-blobs"}`)
				Ω(err).ShouldNot(HaveOccurred())

				// Stores
				err = db.Exec("new-store",
					"32d810b6-073b-4296-8c68-6544a91760f9",
					"redis-shared",
					"Shared Redis services for CF",
					"redis",
					`{"host":"10.9.0.23","port":"6379"}`)
				Ω(err).ShouldNot(HaveOccurred())

				err = db.Exec("new-store",
					"9eb022c4-227f-44f1-b11b-b6d8bcfc3c4f",
					"s3",
					"Amazon S3 Blobstore",
					"s3",
					`{"url":"amazonaws.com","bucket":"bosh-blobs"}`)
				Ω(err).ShouldNot(HaveOccurred())

				// Schedules
				err = db.Exec("new-schedule",
					"b36eeea3-9f5c-46f2-a337-6de5344e8d0f",
					"Weekly Backups",
					"A schedule for weekly bosh-blobs, during normal maintenance windows",
					"sundays at 3:15am")
				Ω(err).ShouldNot(HaveOccurred())

				err = db.Exec("new-schedule",
					"4b8432b9-b5e9-46d5-b23e-ba70983a2acc",
					"Daily Backups",
					"Use for daily (11-something-at-night) bosh-blobs",
					"daily at 11:24pm")
				Ω(err).ShouldNot(HaveOccurred())

				// Retention
				err = db.Exec("new-retention",
					"52c6f512-5c90-4364-998c-f849fa416243",
					"Hourly Retention",
					"Keep backups for 1 hour",
					3600)
				Ω(err).ShouldNot(HaveOccurred())

				err = db.Exec("new-retention",
					"f7dee6c2-59c4-439d-9d92-04046b8beb68",
					"Yearly Retention",
					"Keep backups for 1 year",
					31536000)
				Ω(err).ShouldNot(HaveOccurred())
			})
			It("should return a complete list of jobs", func() {
				jobs, err := orm.GetAllJobs()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(len(jobs)).Should(Equal(2))
			})
		})*/
	})
})