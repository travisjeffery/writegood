(ns user
  (:require
   [clojure.walk :as walk]
   [writegood.schema :as s]
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

(defn q
  [query-string]
  (-> (lacinia/execute schema query-string nil nil)
      simplify))
