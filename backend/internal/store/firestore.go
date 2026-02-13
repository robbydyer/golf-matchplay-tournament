package store

// FirestoreStore is a placeholder for a Google Cloud Firestore-backed implementation
// of the Store interface. To use this, add the following dependency:
//
//   go get cloud.google.com/go/firestore
//
// Then implement each method using the Firestore client.
//
// Example initialization:
//
//   import "cloud.google.com/go/firestore"
//
//   type FirestoreStore struct {
//       client     *firestore.Client
//       collection string
//   }
//
//   func NewFirestoreStore(ctx context.Context, projectID, collection string) (*FirestoreStore, error) {
//       client, err := firestore.NewClient(ctx, projectID)
//       if err != nil {
//           return nil, fmt.Errorf("creating firestore client: %w", err)
//       }
//       return &FirestoreStore{client: client, collection: collection}, nil
//   }
//
// Each method would use client.Collection(collection).Doc(id) for reads/writes,
// mapping models.Tournament to/from Firestore documents.
