package reddit

import (
	"encoding/json"
)

const (
	kindComment    = "t1"
	kindAccount    = "t2"
	kindPost       = "t3"
	kindMessage    = "t4"
	kindSubreddit  = "t5"
	kindAward      = "t6"
	kindListing    = "Listing"
	kindKarmaList  = "KarmaList"
	kindTrophyList = "TrophyList"
	kindUserList   = "UserList"
	kindMore       = "more"
	kindModAction  = "modaction"
)

// thing is an entity on Reddit.
// Its kind reprsents what it is and what is stored in the Data field.
// e.g. t1 = comment, t2 = user, t3 = post, etc.
type thing struct {
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}

type anchor interface {
	After() string
	Before() string
}

// listing holds things coming from the Reddit API.
// It also contains the after/before anchors useful for subsequent requests.
type listing struct {
	things
	after  string
	before string
}

var _ anchor = &listing{}

func (l *listing) After() string {
	return l.after
}

func (l *listing) Before() string {
	return l.before
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (l *listing) UnmarshalJSON(b []byte) error {
	root := new(struct {
		Data struct {
			Things things `json:"children"`
			After  string `json:"after"`
			Before string `json:"before"`
		} `json:"data"`
	})

	err := json.Unmarshal(b, root)
	if err != nil {
		return err
	}

	l.things = root.Data.Things
	l.after = root.Data.After
	l.before = root.Data.Before

	return nil
}

type things struct {
	Comments   []*Comment
	Mores      []*More
	Users      []*User
	Posts      []*Post
	Subreddits []*Subreddit
	ModActions []*ModAction
}

// init initializes or clears the listing.
func (t *things) init() {
	t.Comments = make([]*Comment, 0)
	t.Mores = make([]*More, 0)
	t.Users = make([]*User, 0)
	t.Posts = make([]*Post, 0)
	t.Subreddits = make([]*Subreddit, 0)
	t.ModActions = make([]*ModAction, 0)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (t *things) UnmarshalJSON(b []byte) error {
	t.init()

	var things []thing
	if err := json.Unmarshal(b, &things); err != nil {
		return err
	}

	for _, thing := range things {
		switch thing.Kind {
		case kindComment:
			v := new(Comment)
			if err := json.Unmarshal(thing.Data, v); err == nil {
				t.Comments = append(t.Comments, v)
			}
		case kindMore:
			v := new(More)
			if err := json.Unmarshal(thing.Data, v); err == nil {
				t.Mores = append(t.Mores, v)
			}
		case kindAccount:
			v := new(User)
			if err := json.Unmarshal(thing.Data, v); err == nil {
				t.Users = append(t.Users, v)
			}
		case kindPost:
			v := new(Post)
			if err := json.Unmarshal(thing.Data, v); err == nil {
				t.Posts = append(t.Posts, v)
			}
		case kindSubreddit:
			v := new(Subreddit)
			if err := json.Unmarshal(thing.Data, v); err == nil {
				t.Subreddits = append(t.Subreddits, v)
			}
		case kindModAction:
			v := new(ModAction)
			if err := json.Unmarshal(thing.Data, v); err == nil {
				t.ModActions = append(t.ModActions, v)
			}
		}
	}

	return nil
}

// Comment is a comment posted by a user.
type Comment struct {
	ID      string     `json:"id,omitempty"`
	FullID  string     `json:"name,omitempty"`
	Created *Timestamp `json:"created_utc,omitempty"`
	Edited  *Timestamp `json:"edited,omitempty"`

	ParentID  string `json:"parent_id,omitempty"`
	Permalink string `json:"permalink,omitempty"`

	Body            string `json:"body,omitempty"`
	Author          string `json:"author,omitempty"`
	AuthorID        string `json:"author_fullname,omitempty"`
	AuthorFlairText string `json:"author_flair_text,omitempty"`
	AuthorFlairID   string `json:"author_flair_template_id,omitempty"`

	SubredditName         string `json:"subreddit,omitempty"`
	SubredditNamePrefixed string `json:"subreddit_name_prefixed,omitempty"`
	SubredditID           string `json:"subreddit_id,omitempty"`

	// Indicates if you've upvote/downvoted (true/false).
	// If neither, it will be nil.
	Likes *bool `json:"likes"`

	Score            int `json:"score"`
	Controversiality int `json:"controversiality"`

	PostID string `json:"link_id,omitempty"`
	// This doesn't appear consistently.
	PostTitle string `json:"link_title,omitempty"`
	// This doesn't appear consistently.
	PostPermalink string `json:"link_permalink,omitempty"`
	// This doesn't appear consistently.
	PostAuthor string `json:"link_author,omitempty"`
	// This doesn't appear consistently.
	PostNumComments *int `json:"num_comments,omitempty"`

	IsSubmitter bool `json:"is_submitter"`
	ScoreHidden bool `json:"score_hidden"`
	Saved       bool `json:"saved"`
	Stickied    bool `json:"stickied"`
	Locked      bool `json:"locked"`
	CanGild     bool `json:"can_gild"`
	NSFW        bool `json:"over_18"`

	Replies Replies `json:"replies"`
}

// HasMore determines whether the comment has more replies to load in its reply tree.
func (c *Comment) HasMore() bool {
	return c.Replies.More != nil && len(c.Replies.More.Children) > 0
}

// addCommentToReplies traverses the comment tree to find the one
// that the 2nd comment is replying to. It then adds it to its replies.
func (c *Comment) addCommentToReplies(comment *Comment) {
	if c.FullID == comment.ParentID {
		c.Replies.Comments = append(c.Replies.Comments, comment)
		return
	}

	for _, reply := range c.Replies.Comments {
		reply.addCommentToReplies(comment)
	}
}

func (c *Comment) addMoreToReplies(more *More) {
	if c.FullID == more.ParentID {
		c.Replies.More = more
		return
	}

	for _, reply := range c.Replies.Comments {
		reply.addMoreToReplies(more)
	}
}

// Replies holds replies to a comment.
// It contains both comments and "more" comments, which are entrypoints to other
// comments that were left out.
type Replies struct {
	Comments []*Comment `json:"comments,omitempty"`
	More     *More      `json:"-"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *Replies) UnmarshalJSON(data []byte) error {
	// if a comment has no replies, its "replies" field is set to ""
	if string(data) == `""` {
		r = nil
		return nil
	}

	root := new(listing)
	err := json.Unmarshal(data, root)
	if err != nil {
		return err
	}

	r.Comments = root.Comments
	if len(root.Mores) > 0 {
		r.More = root.Mores[0]
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (r *Replies) MarshalJSON() ([]byte, error) {
	if r == nil || len(r.Comments) == 0 {
		return []byte(`null`), nil
	}
	return json.Marshal(r.Comments)
}

// More holds information used to retrieve additional comments omitted from a base comment tree.
type More struct {
	ID       string `json:"id"`
	FullID   string `json:"name"`
	ParentID string `json:"parent_id"`
	// Total number of replies to the parent + replies to those replies (recursively).
	Count int `json:"count"`
	// Number of comment nodes from the parent down to the furthest comment node.
	Depth    int      `json:"depth"`
	Children []string `json:"children"`
}

// Post is a submitted post on Reddit.
type Post struct {
	ID      string     `json:"id,omitempty"`
	FullID  string     `json:"name,omitempty"`
	Created *Timestamp `json:"created_utc,omitempty"`
	Edited  *Timestamp `json:"edited,omitempty"`

	Permalink string `json:"permalink,omitempty"`
	URL       string `json:"url,omitempty"`

	Title string `json:"title,omitempty"`
	Body  string `json:"selftext,omitempty"`

	// Indicates if you've upvote/downvoted (true/false).
	// If neither, it will be nil.
	Likes *bool `json:"likes"`

	Score            int     `json:"score"`
	UpvoteRatio      float32 `json:"upvote_ratio"`
	NumberOfComments int     `json:"num_comments"`

	SubredditName         string `json:"subreddit,omitempty"`
	SubredditNamePrefixed string `json:"subreddit_name_prefixed,omitempty"`
	SubredditID           string `json:"subreddit_id,omitempty"`

	Author   string `json:"author,omitempty"`
	AuthorID string `json:"author_fullname,omitempty"`

	Spoiler    bool `json:"spoiler"`
	Locked     bool `json:"locked"`
	NSFW       bool `json:"over_18"`
	IsSelfPost bool `json:"is_self"`
	Saved      bool `json:"saved"`
	Stickied   bool `json:"stickied"`
}

// Subreddit holds information about a subreddit
type Subreddit struct {
	ID      string     `json:"id,omitempty"`
	FullID  string     `json:"name,omitempty"`
	Created *Timestamp `json:"created_utc,omitempty"`

	URL                  string `json:"url,omitempty"`
	Name                 string `json:"display_name,omitempty"`
	NamePrefixed         string `json:"display_name_prefixed,omitempty"`
	Title                string `json:"title,omitempty"`
	Description          string `json:"public_description,omitempty"`
	Type                 string `json:"subreddit_type,omitempty"`
	SuggestedCommentSort string `json:"suggested_comment_sort,omitempty"`

	Subscribers     int  `json:"subscribers"`
	ActiveUserCount *int `json:"active_user_count,omitempty"`
	NSFW            bool `json:"over18"`
	UserIsMod       bool `json:"user_is_moderator"`
	Subscribed      bool `json:"user_is_subscriber"`
	Favorite        bool `json:"user_has_favorited"`
}

// PostAndComments is a post and its comments.
type PostAndComments struct {
	Post     *Post      `json:"post"`
	Comments []*Comment `json:"comments"`
	More     *More      `json:"-"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// When getting a sticky post, you get an array of 2 Listings
// The 1st one contains the single post in its children array
// The 2nd one contains the comments to the post
func (pc *PostAndComments) UnmarshalJSON(data []byte) error {
	var l [2]listing

	err := json.Unmarshal(data, &l)
	if err != nil {
		return err
	}

	pc.Post = l[0].Posts[0]
	pc.Comments = l[1].Comments
	if len(l[1].Mores) > 0 {
		pc.More = l[1].Mores[0]
	}

	return nil
}

// HasMore determines whether the post has more replies to load in its reply tree.
func (pc *PostAndComments) HasMore() bool {
	return pc.More != nil && len(pc.More.Children) > 0
}

func (pc *PostAndComments) addCommentToTree(comment *Comment) {
	if pc.Post.FullID == comment.ParentID {
		pc.Comments = append(pc.Comments, comment)
		return
	}

	for _, reply := range pc.Comments {
		reply.addCommentToReplies(comment)
	}
}

func (pc *PostAndComments) addMoreToTree(more *More) {
	if pc.Post.FullID == more.ParentID {
		pc.More = more
	}

	for _, reply := range pc.Comments {
		reply.addMoreToReplies(more)
	}
}
