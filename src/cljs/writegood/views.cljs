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
      [:trix-editor])))

(defn main-panel []
  [textarea])
