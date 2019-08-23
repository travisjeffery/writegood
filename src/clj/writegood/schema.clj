(ns writegood.schema
  (:require
   [clojure.java.io :as io]
   [writegood.db :as db]
   [com.walmartlabs.lacinia.util :as util]
   [com.walmartlabs.lacinia.schema :as schema]
   [com.walmartlabs.lacinia.resolve :refer [resolve-as]]
   [com.stuartsierra.component :as component]
   [clojure.edn :as edn]))

(defn user-documents
  [db]
  (fn [_ _ user]
    (db/list-documents-for-user db (:id user))))

(defn user-by-id
  [db]
  (fn [_ args _]
    (db/find-user-by-id db (:id args))))

(defn entity-map
  [data k]
  (reduce #(assoc %1 (:id %2) %2)
          {}
          (get data k)))

(defn upsert-document
  [db]
  (fn [_ args _]
    (let [{document-id :id
           author-id :author_id
           text :text} args
          document (db/find-document-by-id db document-id)]
      (cond
        (nil? author-id)
        (resolve-as nil {:message "Author not found."
                         :status 404})

        (nil? text)
        (resolve-as nil {:message "Text must be non-empty."
                         :status 400})

        :else
        (do
          (db/upsert-document db author-id document-id text)
          document)))))

(defn resolver-map
  [component]
  (let [db (:db component)]
    {:query/user-by-id (user-by-id db)
     :query/documents-by-user (user-documents db)
     :mutation/upsert-document (upsert-document db)}))

(defn load-schema
  [component]
  (-> (io/resource "schema.edn")
      slurp
      edn/read-string
      (util/attach-resolvers (resolver-map component))
      schema/compile))

(defrecord SchemaProvider
           [schema]

  component/Lifecycle
  (start [this]
    (assoc this :schema (load-schema this)))

  (stop [this]
    (assoc this :schema nil)))

(defn new-schema-provider
  []
  {:schema-provider (->  {}
                         map->SchemaProvider
                         (component/using [:db]))})
