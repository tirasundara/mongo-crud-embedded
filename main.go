package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const uri = "mongodb://localhost:27017"

func main() {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}
	db := client.Database("cruds")

	defer func() {
		if err := db.Drop(context.TODO()); err != nil {
			panic(err)
		}

		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}

		fmt.Println("Done. Bye!")
	}()

	coll := db.Collection("users")
	store := &UserStore{coll: coll}

	u, err := store.Insert(context.TODO(), User{Username: "tira", Posts: []Post{{ID: primitive.NewObjectID().Hex(), Title: "aaaa", Body: "abccdcasc", Likes: 999}}})
	if err != nil {
		log.Fatalf("store.Insert: %v", err)
	}

	fmt.Printf("user: %v\n", u)
	fmt.Printf("user posts: %v\n", u.Posts)

	anotherPost := Post{ID: primitive.NewObjectID().Hex(), Title: "bbbb", Body: "qqqqqq", Likes: 1111}
	anotherPost2 := Post{ID: primitive.NewObjectID().Hex(), Title: "cccc", Body: "wwwwww", Likes: 2222}
	if err = store.AddPosts(context.TODO(), u.ID, anotherPost, anotherPost2); err != nil {
		log.Fatalf("store.AddPosts: %v", err)
	}

	u, err = store.FindByID(context.TODO(), u.ID)
	if err != nil {
		log.Fatalf("store.FindByID: %v", u.ID)
	}

	fmt.Printf("user: %v\n", u)
	fmt.Printf("user posts: %v\n", u.Posts)

	anotherPost2.Title = "my updated title"
	anotherPost2.Body = "my updated body"
	if err := store.UpdatePost(context.TODO(), u.ID, anotherPost2.ID, anotherPost2); err != nil {
		if err.Error() != "no document modified" {
			log.Fatalf("store.UpdatePost: %v", err)
		}
	}

	u, err = store.FindByID(context.TODO(), u.ID)
	if err != nil {
		log.Fatalf("store.FindByID: %v", u.ID)
	}

	fmt.Printf("user: %v\n", u)
	fmt.Printf("user posts: %v\n", u.Posts)

	if err := store.DeletePost(context.TODO(), u.ID, anotherPost2.ID); err != nil {
		log.Fatalf("store.DeletePost: %v", err)
	}

	u, err = store.FindByID(context.TODO(), u.ID)
	if err != nil {
		log.Fatalf("store.FindByID: %v", u.ID)
	}

	fmt.Printf("user: %v\n", u)
	fmt.Printf("user posts: %v\n", u.Posts)

}

type User struct {
	ID       string `bson:"_id,omitempty"`
	Username string `bson:"username"`
	Posts    []Post `bson:"posts"`
}

type Post struct {
	ID    string `bson:"_id,omitempty"`
	Title string `bson:"title"`
	Body  string `bson:"body"`
	Likes uint   `bson:"likes"`
}

type Storer interface {
	Insert(ctx context.Context, user User) (User, error)
	FindByID(ctx context.Context, id string) (User, error)
	// Update(ctx context.Context, user User) (User, error)
	// DeleteByID(ctx context.Context, id string) error
	AddPosts(ctx context.Context, userID string, post ...Post) error
	UpdatePost(ctx context.Context, userID, postID string, post Post) error
	DeletePost(ctx context.Context, userID, postID string) error
}

type UserStore struct {
	coll *mongo.Collection
}

func (s *UserStore) Insert(ctx context.Context, user User) (User, error) {
	r, err := s.coll.InsertOne(ctx, user)
	if err != nil {
		return user, err
	}

	user.ID = r.InsertedID.(primitive.ObjectID).Hex()
	return user, nil
}

func (s *UserStore) FindByID(ctx context.Context, id string) (User, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return User{}, err
	}

	filter := bson.M{"_id": objID}
	var user User
	err = s.coll.FindOne(ctx, filter).Decode(&user)
	return user, err
}

func (s *UserStore) AddPosts(ctx context.Context, userID string, posts ...Post) error {
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}
	filter := bson.M{"_id": objID}
	changeset := bson.M{"$push": bson.M{"posts": bson.M{"$each": posts}}}

	r, err := s.coll.UpdateOne(ctx, filter, changeset)
	if err != nil {
		return err
	}

	if r.ModifiedCount == 0 {
		return errors.New("no document modified")
	}

	return nil
}

func (s *UserStore) UpdatePost(ctx context.Context, userID, postID string, post Post) error {
	objUserID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": objUserID, "posts._id": postID}
	changeset := bson.M{"$set": bson.M{"posts.$": post}}

	r, err := s.coll.UpdateOne(ctx, filter, changeset)
	if err != nil {
		return err
	}

	if r.ModifiedCount == 0 {
		return errors.New("no document modified")
	}

	return nil
}

func (s *UserStore) DeletePost(ctx context.Context, userID, postID string) error {
	objUserID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}
	filter := bson.M{"_id": objUserID}
	changeset := bson.M{"$pull": bson.M{"posts": bson.M{"_id": postID}}}

	r, err := s.coll.UpdateOne(ctx, filter, changeset)
	if err != nil {
		return err
	}

	if r.ModifiedCount == 0 {
		return errors.New("no document modified")
	}

	return nil
}
