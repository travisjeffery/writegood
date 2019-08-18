(ns writegood.events
  (:require
   [re-frame.core :as re-frame]
   [writegood.db :as db]
   ))

(re-frame/reg-event-db
 ::initialize-db
 (fn [_ _]
   db/default-db))


(re-frame/reg-event-db
 ::change-text
 (fn [db [_ new]]
   (assoc db :text new)))
