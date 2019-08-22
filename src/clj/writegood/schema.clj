(ns writegood.schema
  (:require
   [clojure.java.io :as io]
   [com.walmartlabs.lacinia.util :as util]
   [com.walmartlabs.lacinia.schema :as schema]
   [clojure.edn :as edn]))

(defn resolve-user-by-id
  [users-map context args value]
  (let [{:keys [id]} args]
    (get users-map id)))

(defn resolve-document-by-user
  [documents-map context args user]
  (->> documents-map
       vals
       (filter #(-> % :author (= (:id user))))))

(defn entity-map
  [data k]
  (reduce #(assoc %1 (:id %2) %2)
          {}
          (get data k)))

(defn resolver-map
  []
  (let [data (-> (io/resource "data.edn")
                 slurp
                 edn/read-string)
        users-map (entity-map data :users)
        documents-map (entity-map data :documents)]
    {:query/user-by-id (partial resolve-user-by-id users-map)
     :query/documents-by-user (partial resolve-document-by-user documents-map)}))
(defn load-schema
  []
  (-> (io/resource "schema.edn")
      slurp
      edn/read-string
      (util/attach-resolvers (resolver-map))
      schema/compile))
