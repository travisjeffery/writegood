(ns writegood.views
  (:require
   [reagent.core :as r]
   [re-frame.core :as re-frame]
   [writegood.subs :as subs]
   [writegood.events :as events]))

(def editor-versions (r/atom '()))

(def editor-text (r/atom ""))

(defn textarea []
  [:form
   [:input#x {:value @editor-text :type "hidden" :name "content"}]
   [editor-toolbar]
   [:trix-editor#editor {:toolbar "toolbar" :input "x" :placeholder "Write..."}]])

(defn editor-change [event]
  (let [text (-> event .-target .-textContent)]
    (reset! editor-text text)))

(defn setup-editor []
  (.addEventListener (.getElementById js/document "editor") "trix-change" editor-change false))

(defn main-panel []
  [textarea])
