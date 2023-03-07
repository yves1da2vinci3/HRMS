package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

var mg MongoInstance

const dbName = "fiberHrms"
const mongoURI = "mongodb+srv://nodeuser:nodejsissocool@cluster0.1l7ra.mongodb.net/" + dbName

type Employee struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty" `
	Name   string  `json:"name"`
	Salary float64 `json:"salary"`
	Age    float64 `json:"age"`
}

func Connect() error {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	db := client.Database(dbName)
	if err != nil {
		log.Fatal(err)
	}
	mg = MongoInstance{Client: client, Db: db}
	return nil
}

func main() {
	if err := Connect(); err != nil {
		log.Fatal(err)
	}
	app := fiber.New()
	app.Get("/employee", func(c *fiber.Ctx) error {
		query := bson.D{{}}
		cursor, err := mg.Db.Collection("employees").Find(c.Context(), query)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		var employees []Employee = make([]Employee, 0)
		if err := cursor.All(c.Context(), &employees); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}
		return c.JSON(employees)
	})
	app.Post("/employee", func(c *fiber.Ctx) error {
		collection := mg.Db.Collection("employees")
		employee := new(Employee)
		if err := c.BodyParser(employee); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString(err.Error())
		}
		employee.ID = ""
		insertionResult, err := collection.InsertOne(c.Context(), employee)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		filter := bson.D{{Key: "_id", Value: insertionResult.InsertedID}}
		createRecord := collection.FindOne(c.Context(), filter)

		createEmployee := &Employee{}
		createRecord.Decode(createEmployee)
		return c.Status(fiber.StatusCreated).JSON(createEmployee)

	})
	app.Put("/employee/:id", func(c *fiber.Ctx) error {
		idParam := c.Params("id")
		employeeID, err := primitive.ObjectIDFromHex(idParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString(err.Error())
		}
		employee := new(Employee)
		if err := c.BodyParser(employee); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString(err.Error())
		}

		query := bson.D{{Key: "_id", Value: employeeID}}
		update := bson.D{
			{
				Key: "$set",
				Value: bson.D{
					{Key: "name", Value: employee.Name},
					{Key: "salary", Value: employee.Salary},
					{Key: "age", Value: employee.Age},
				},
			},
		}
		err = mg.Db.Collection("employees").FindOneAndUpdate(c.Context(), query, update).Err()
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return c.Status(404).SendString("Employee not found")
			}
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())

		}
		employee.ID = idParam
		return c.Status(fiber.StatusCreated).JSON(employee)
	})

	app.Delete("/employee/:id", func(c *fiber.Ctx) error {

		employeeID, Err := primitive.ObjectIDFromHex(
			c.Params("id"),
		)
		if Err != nil {
			return c.Status(fiber.StatusBadRequest).SendString(Err.Error())
		}
		query := bson.D{{Key: "_id", Value: employeeID}}
		result, err := mg.Db.Collection("employees").DeleteOne(c.Context(), &query)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).Send([]byte(err.Error()))
		}
		if result.DeletedCount < 1 {
			return c.Status(fiber.StatusNotFound).Send([]byte(err.Error()))
		}

		return c.Status(fiber.StatusAccepted).JSON("record deleted")
	})

	log.Fatal(app.Listen(":3000"))
}
