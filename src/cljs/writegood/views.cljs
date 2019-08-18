(ns writegood.views
  (:require
   [reagent.core :as r]
   [re-frame.core :as re-frame]
   [writegood.subs :as subs]
   [writegood.events :as events]))

(def editor-text (r/atom ""))

(defn editor-toolbar []
  [:trix-toolbar#toolbar
   [:div {:class "trix-button-row"}
    [:span {:class "trix-button-group trix-button-group--text-tools", :data-trix-button-group "text-tools"}
     [:button {:type "button", :class "trix-button trix-button--icon trix-button--icon-bold", :data-trix-attribute "bold", :data-trix-key "b", :title "#{lang.bold}", :tabindex "-1"} "#{lang.bold}"]
     [:button {:type "button", :class "trix-button trix-button--icon trix-button--icon-italic", :data-trix-attribute "italic", :data-trix-key "i", :title "#{lang.italic}", :tabindex "-1"} "#{lang.italic}"]
     [:button {:type "button", :class "trix-button trix-button--icon trix-button--icon-strike", :data-trix-attribute "strike", :title "#{lang.strike}", :tabindex "-1"} "#{lang.strike}"]
     [:button {:type "button", :class "trix-button trix-button--icon trix-button--icon-link", :data-trix-attribute "href", :data-trix-action "link", :data-trix-key "k", :title "#{lang.link}", :tabindex "-1"} "#{lang.link}"]]
    [:span {:class "trix-button-group trix-button-group--block-tools", :data-trix-button-group "block-tools"}
     [:button {:type "button", :class "trix-button trix-button--icon trix-button--icon-heading-1", :data-trix-attribute "heading1", :title "#{lang.heading1}", :tabindex "-1"} "#{lang.heading1}"]
     [:button {:type "button", :class "trix-button trix-button--icon trix-button--icon-quote", :data-trix-attribute "quote", :title "#{lang.quote}", :tabindex "-1"} "#{lang.quote}"]
     [:button {:type "button", :class "trix-button trix-button--icon trix-button--icon-code", :data-trix-attribute "code", :title "#{lang.code}", :tabindex "-1"} "#{lang.code}"]
     [:button {:type "button", :class "trix-button trix-button--icon trix-button--icon-bullet-list", :data-trix-attribute "bullet", :title "#{lang.bullets}", :tabindex "-1"} "#{lang.bullets}"]
     [:button {:type "button", :class "trix-button trix-button--icon trix-button--icon-number-list", :data-trix-attribute "number", :title "#{lang.numbers}", :tabindex "-1"} "#{lang.numbers}"]
     [:button {:type "button", :class "trix-button trix-button--icon trix-button--icon-decrease-nesting-level", :data-trix-action "decreaseNestingLevel", :title "#{lang.outdent}", :tabindex "-1"} "#{lang.outdent}"]
     [:button {:type "button", :class "trix-button trix-button--icon trix-button--icon-increase-nesting-level", :data-trix-action "increaseNestingLevel", :title "#{lang.indent}", :tabindex "-1"} "#{lang.indent}"]]
    [:span {:class "trix-button-group trix-button-group--file-tools", :data-trix-button-group "file-tools"}
     [:button {:type "button", :class "trix-button trix-button--icon trix-button--icon-attach", :data-trix-action "attachFiles", :title "#{lang.attachFiles}", :tabindex "-1"} "#{lang.attachFiles}"]]
    [:span {:class "trix-button-group-spacer"}]
    [:span {:class "trix-button-group trix-button-group--history-tools", :data-trix-button-group "history-tools"}
     [:button {:type "button", :class "trix-button trix-button--icon trix-button--icon-undo", :data-trix-action "undo", :data-trix-key "z", :title "#{lang.undo}", :tabindex "-1"} "#{lang.undo}"]
     [:button {:type "button", :class "trix-button trix-button--icon trix-button--icon-redo", :data-trix-action "redo", :data-trix-key "shift+z", :title "#{lang.redo}", :tabindex "-1"} "#{lang.redo}"]]]
   [:div {:class "trix-dialogs", :data-trix-dialogs true}
    [:div {:class "trix-dialog trix-dialog--link", :data-trix-dialog "href", :data-trix-dialog-attribute "href"}
     [:div {:class "trix-dialog__link-fields"}
      [:input {:type "url", :name "href", :class "trix-input trix-input--dialog", :placeholder "#{lang.urlPlaceholder}", :aria-label "#{lang.url}", :required , :data-trix-input}]
      [:div {:class "trix-button-group"}
       [:input {:type "button", :class "trix-button trix-button--dialog", :value "#{lang.link}", :data-trix-method "setAttribute"}]
       [:input {:type "button", :class "trix-button trix-button--dialog", :value "#{lang.unlink}", :data-trix-method "removeAttribute"}]]]]]])

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
