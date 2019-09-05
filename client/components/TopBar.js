import React from 'react'
import { TopBar as PolarisTopBar, Frame, Page, Card, ActionList, Layout } from '@shopify/polaris'

class TopBar extends React.Component {
  state = {
    userMenuOpen: false,
    searchActive: false,
    searchText: '',
  }

  render() {
    const {
      state,
      handleSearchChange,
      handleSearchResultsDismiss,
      toggleUserMenu,
    } = this;
    const {userMenuOpen, searchText, searchActive} = state;

    const userMenuMarkup = (
      <PolarisTopBar.UserMenu
        actions={[
          {
            items: [{content: 'Back to Shopify'}],
          },
          {
            items: [{content: 'Community forums'}],
          },
        ]}
        name="tj@travisjeffery.com"
        initials="T"
        open={userMenuOpen}
        onToggle={toggleUserMenu}
      />
    );

    const searchResultsMarkup = (
      <Card>
        <ActionList
          items={[
            {content: 'Shopify help center'},
            {content: 'Community forums'},
          ]}
        />
      </Card>
    );

    const searchFieldMarkup = (
      <PolarisTopBar.SearchField
        onChange={handleSearchChange}
        value={searchText}
        placeholder="Search"
      />
    );

    return (
      <PolarisTopBar
        showNavigationToggle={true}
        userMenu={userMenuMarkup}
        searchResultsVisible={searchActive}
        searchField={searchFieldMarkup}
        searchResults={searchResultsMarkup}
        onSearchResultsDismiss={handleSearchResultsDismiss}
        onNavigationToggle={() => {
          console.log('toggle navigation visibility');
        }}
      />
    );
  }

  toggleUserMenu = () => {
    this.setState(({userMenuOpen}) => ({userMenuOpen: !userMenuOpen}));
  };

  handleSearchResultsDismiss = () => {
    this.setState(() => {
      return {
        searchActive: false,
        searchText: '',
      };
    });
  };

  handleSearchChange = (value) => {
    this.setState({searchText: value});
    if (value.length > 0) {
      this.setState({searchActive: true});
    } else {
      this.setState({searchActive: false});
    }
  };
}

export default TopBar
