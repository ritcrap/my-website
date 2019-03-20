import React from 'react';

import SearchBox from '../components/SearchBox';
import ContentBox from '../components/ContentBox';

const HomePage = () => {
	return (
		<div className="HomePage-c-00">
			<div className="HomePage-Container-00">
				<h1>Api Management Console</h1>
				<div style={{ alignItems: 'flex-start' }} className="Flex-Row-Space-Between">
					<SearchBox />
					<ContentBox />
				</div>
			</div>
		</div>
	);
};

export default HomePage;