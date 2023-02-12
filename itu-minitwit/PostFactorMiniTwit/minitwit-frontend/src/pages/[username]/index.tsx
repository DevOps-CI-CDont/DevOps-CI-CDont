import { TweetContainer } from "@/components/Message/TweetContainer";
import DefaultLayout from "@/layouts/DefaultLayout";
import { getPublicTweets } from "@/server/getPublicTweets";
import { Tweet } from "@/types/tweet.type";

interface UserProfileProps {
  username?: string;
  tweets?: Tweet[];
}

export default function UserProfilePage({
  username,
  tweets,
}: UserProfileProps) {
  return (
    <DefaultLayout>
      <div className='mt-4'>
        <h1>{username}&quot;s profile</h1>
        <TweetContainer tweets={tweets} />
      </div>
    </DefaultLayout>
  );
}

export async function getServerSideProps(context: any) {
  try {
    const { username } = context.query;
    console.log(username);

    const messages = await getPublicTweets();

    if (!messages.tweets) {
      throw new Error("No tweets found");
    }

    const filteredTweets = messages.tweets.filter(
      (tweet: Tweet) => tweet.author.username === username
    );

    console.log(filteredTweets);

    return {
      props: {
        username,
        tweets: filteredTweets,
      },
    };
  } catch (e) {
    return {
      notFound: true,
    };
  }
}
