(ns writegood.system-test
  (:require
   [clojure.test :refer [deftest is]]
   [writegood.system :as system]
   [writegood.test-utils :refer [simplify]]
   [com.stuartsierra.component :as component]
   [com.walmartlabs.lacinia :as lacinia]))

(defn ^:private test-system
  "Creates a new system suitable for testing, and ensures that the HTTP port won't conflict with a default running system."
  []
  (-> (system/new-system)
      (assoc-in [:server :port] 8989)))

(defn ^:private q
  [system query variables]
  (-> system
      (get-in [:schema-provider :schema])
      (lacinia/execute query variables nil)
      simplify))

(deftest can-read-documents
  (let [system (component/start-system (test-system))
        results (q system
                   " { user_by_id(id: 1) { id email documents { id text } } }"
                   nil)]
    (is (= {:data {:user_by_id {:id 1
                                :email "tj@travisjeffery.com"
                                :documents [{:id 1 :text "TJs note."}]}}}
           results))
    (component/stop-system system)))
