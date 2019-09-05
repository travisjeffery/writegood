import React from 'react'
import { Editor as Slate } from 'slate-react'

const Editor = ({editor, changeEditor}) => (
    <Slate value={editor} onChange={changeEditor} />
)

export default Editor
