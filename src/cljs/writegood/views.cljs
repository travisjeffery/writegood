(ns writegood.views
  (:require
   [reagent.core :as r]
   [re-frame.core :as re-frame]
   [writegood.subs :as subs]
   [writegood.events :as events]
   ))

(defn debugger [& args]
  (js/eval "debugger"))

(defn textarea []
  (let [text (r/atom "")]
    (fn []
      [:textarea {:value @text
                  :on-change #(reset! text (-> % .-target .-value))
                  :placeholder "Write..."}])))

(defn main-panel []
  [textarea])
