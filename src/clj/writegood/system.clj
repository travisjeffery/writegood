(ns writegood.system
  (:require
   [com.stuartsierra.component :as component]
   [writegood.schema :as schema]
   [writegood.db :as db]
   [writegood.server :as server]))

(defn new-system
  []
  (merge (component/system-map)
         (server/new-server)
         (schema/new-schema-provider)
         (db/new-db)))
