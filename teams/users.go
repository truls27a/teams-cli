package teams

import "context"

type User struct {
	MRI               string `json:"mri"`
	ObjectID          string `json:"objectId"`
	DisplayName       string `json:"displayName"`
	GivenName         string `json:"givenName"`
	Surname           string `json:"surname"`
	Email             string `json:"email"`
	UserPrincipalName string `json:"userPrincipalName"`
	UserType          string `json:"userType"`
	TenantName        string `json:"tenantName"`
	IsShortProfile    bool   `json:"isShortProfile"`
	Type              string `json:"type"`
}

type usersResponse struct {
	Type  string `json:"type"`
	Value []User `json:"value"`
}

func (c *Client) FetchShortProfile(ctx context.Context, mris []string) ([]User, error) {
	if len(mris) == 0 {
		return nil, nil
	}
	var resp usersResponse
	if err := c.doMT(ctx, "POST", "/users/fetchShortProfile", mris, &resp); err != nil {
		return nil, err
	}
	return resp.Value, nil
}
