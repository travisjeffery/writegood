import React from 'react'
import Document from './Document'

const DocumentList = ({documents, style}) => (
  <ul style={style}>
    {documents.map(document =>
      <Document {...document} />
    )}
  </ul>
)

export default DocumentList
