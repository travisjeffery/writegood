import { connect } from 'react-redux'
import DocumentList from '../components/DocumentList'

const mapStateToProps = state => ({
  documents: state.documents
})

const mapDispatchToProps = dispatch => ({
})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(DocumentList)
