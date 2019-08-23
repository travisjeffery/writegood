(ns writegood.db
  (:require
   [clojure.edn :as edn]
   [clojure.java.io :as io]
   [com.stuartsierra.component :as component]))

(defrecord Database [data]

  component/Lifecycle
  (start [this]
    (assoc this :data (-> (io/resource "data.edn")
                          slurp
                          edn/read-string
                          atom)))
  (stop [this]
    (assoc this :data nil)))

(defn new-db
  []
  {:db (map->Database {})})

(defn find-user-by-id
  [db user-id]
  (->> db
       :data
       deref
       :users
       (filter #(= user-id (:id %)))
       first))

(defn find-document-by-id
  [db document-id]
  (->> db
       :data
       deref
       :documents
       (filter #(= document-id (:id %)))
       first))

(defn ^:private apply-document
  [documents author-id document-id text]
  (->> documents
       (remove #(and (= document-id (:id %))
                     (= author-id (:author_id %))))
       (cons {:id document-id :author_id author-id :text text})))

(defn upsert-document
  "Adds or update a document."
  [db author-id document-id text]
  (-> db
      :data
      (swap! update :documents apply-document author-id document-id text)))

(defn list-documents-for-user
  [db user-id]
  (->> db
       :data
       deref
       :documents
       (filter #(= user-id (:author_id %)))))
