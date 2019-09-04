import { connect } from 'react-redux'
import Editor from '../components/Editor'
import { changeEditor } from '../actions'

const mapStateToProps = state => ({
  editor: state.editor
})

const mapStateToDispatch = dispatch => ({
  changeEditor: value => dispatch(changeEditor(value))
})

export default connect(
  mapStateToProps,
  mapStateToDispatch
)(Editor)
