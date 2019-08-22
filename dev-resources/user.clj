(ns user
  (:require
   [clojure.walk :as walk]
   [writegood.schema :as s]
   [com.walmartlabs.lacinia.pedestal :as lp]
   [io.pedestal.http :as http]
   [clojure.java.browse :refer [browse-url]]
   [com.walmartlabs.lacinia :as lacinia])
  (:import
   (clojure.lang IPersistentMap)))

(def schema (s/load-schema))

(defn simplify
  "Converts all ordered maps nested within the map into standard hash maps and sequences into
  vectors, which makes for easier constansts in the tests, and eliminates ordering problems."
  [m]
  (walk/postwalk
   (fn [node]
     (cond
       (instance? IPersistentMap node)
       (into {} node)
       (seq? node)
       (vec node)
       :else
       node))
   m))

(defonce server nil)

(defn start-server
  [_]
  (let [server (-> schema
                   (lp/service-map {:graphiql true})
                   http/create-server
                   http/start)]
    (browse-url "http://localhost:8888/")
    server))

(defn stop-server
  [server]
  (http/stop server)
  nil)

(defn start
  []
  (alter-var-root #'server start-server)
  :started)

(defn stop
  []
  (alter-var-root #'server stop-server)
  :stopped)

(defn q
  [query-string]
  (-> (lacinia/execute schema query-string nil nil)
      simplify))
