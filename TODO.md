# Required before soft launch
- [x] Google analytics
- [x] Less shit landing page
- [ ] More clear help text throughout
- [x] Handle ? and about buttons
- [ ] Make process of configuring and building a playlist more clear
- [x] Minimize tailwind css / dev/prod build-pipeline/makefile system
- [x] Handle mobile case
- [x] Form validation
- [x] Name and logo
- [x] HTTPS (This is a nightmare)
- [x] Better copy on front page
- [x] Grey background all the way on dashboard page
- [x] Mobile styling for home page
- [x] Don't error on trying to pull too many songs from a list
- [x] Remove use cases and fix laptop stylings

# Long-term goals
- Mobile styling for rest of website
- Table styling for playlist stuff
- GIF on landing page demonstrating how application works
- Make the logo's 'a' and 'p' in 'Mixtape' a cassete
- Make sure name MixtapeManager is used everywhere
- Lock out the heroku subdomain - it is a security risk
- Integration tests
- 404 not found bug when adding a new source (e.g. lofi hip hop beats to study to)
- Add a special page for 404s
- Track sources and build errors should be their own rows in Postgres
- Figure out the freaking asset pipeline
- Setup some CI/CD and stop pushing to master like a savage
- Setup dev environment on heroku

# Questions
- Should build occur as soon as a scheduled playlist has been built? -> No, one manual build required
- What should the behaviour be for counts that are too large for a playlist?
- Should I bump contrast on landing page waves -> No
