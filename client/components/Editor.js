import React from 'react'
import { Editor as Slate } from 'slate-react'

const Editor = ({editor}) => (
  <Slate value={editor} />
)

export default Editor
