import { connect } from 'react-redux'
import Editor from '../components/Editor'

const mapStateToProps = state => ({
  editor: state.editor
})

const mapStateToDispatch = dispatch => ({})

export default connect(
  mapStateToProps,
  mapStateToDispatch
)(Editor)
