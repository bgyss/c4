// github.com/Avalanche-io/c4/store is a package for representing generic C4
// storage. A C4 store abstracts away the details of data management,
// allowing C4 data consumers and producers to store and retrieve
// C4-identified data using the C4 ID alone.
//
// A C4 store could represent an object storage bucket, a local filesystem,
// or the aggregation of many C4 stores. A C4 store can also be used to abstract
// processes like encryption, creating distributed copies, and on-the-fly
// validation.
package store
