(ns writegood.db
  (:require
   [clojure.edn :as edn]
   [io.pedestal.log :as log]
   [clojure.java.jdbc :as jdbc]
   [clojure.java.io :as io]
   [clojure.string :as str]
   [com.stuartsierra.component :as component])
  (:import (com.mchange.v2.c3p0 ComboPooledDataSource)))

(defn ^:private pooled-data-resource
  [host dbname user password port]
  {:datasource
   (doto (ComboPooledDataSource.)
     (.setDriverClass "org.postgresql.Driver")
     (.setJdbcUrl (str "jdbc:postgresql://" host ":" port "/" dbname))
     (.setUser user)
     (.setPassword password))})

(defrecord Database [ds]
  component/Lifecycle
  (start [this]
    (assoc this :ds (pooled-data-resource "localhost" "writegooddb" "writegood_role" "lacinia" 25432)))
  (stop [this]
    (-> ds :datasource .close)
    (assoc this :ds nil)))

(defn new-db
  "Returns a new database."
  []
  {:db (map->Database {})})

(defn find-user-by-id
  [component user-id]
  (first
   (query component
               ["select id, email from users where id = $1", user-id])))

  (defn find-document-by-id
  [component document-id]
  (first
   (query component
               ["select id, text from documents where id = $1" document-id])))

(defn ^:private apply-document
  [component author-id document-id text]
  (first
   (query component
               (if document-id
                 ["update documents set text = $1 where id = $2 RETURNING id, author_id text" text document-id]
                 ["insert into documents (author_id, text) values ($1, $2) RETURNING id, author_id, text" author-id text]))))

(defn ^:private query
  [component statement]
  (let [[sql & params] statement]
    (log/debug :sql (str/replace sql #"\s+" " ")
               :params params))
  (jdbc/query (:ds component) statement))

(defn upsert-document
  "Adds or update a document."
  [component author-id document-id text]
  (apply-document component author-id document-id text))

(defn list-documents-for-author
  [component author-id]
  (query component
              ["select id, author_id, text from documents where author_id = $1" author-id]))
