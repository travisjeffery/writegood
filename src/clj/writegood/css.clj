(ns writegood.css
  (:require [garden.def :refer [defstyles]]))

(defstyles screen
  [:textarea {:width "99vw" :height "97.3vh"}]

  [:.text-button {:background "none"
                  :margin "0"
                  :padding "0"
                  :border "none"
                  :cursor "pointer"}])
