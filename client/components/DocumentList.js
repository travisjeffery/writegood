import React from 'react'
import Document from './Document'

const DocumentList = ({documents}) => (
  <ul>
    {documents.map(document =>
      <Document {...document} />
    )}
  </ul>
)

export default DocumentList
